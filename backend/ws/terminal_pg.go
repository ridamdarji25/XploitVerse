package ws

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xploitverse/backend/internal/config"
	"github.com/xploitverse/backend/internal/utils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func sendError(conn *websocket.Conn, msg string) {
	_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mError: "+msg+"\033[0m\r\n"))
}

// TerminalHandlerPG bridges browser WebSocket terminal traffic to a running
// Docker container for PostgreSQL-backed lab sessions.
func TerminalHandlerPG(db *pgxpool.Pool, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			hdr := c.GetHeader("Authorization")
			tokenStr = strings.TrimPrefix(hdr, "Bearer ")
		}
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		claims, err := utils.VerifyToken(tokenStr, cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		userID, err := strconv.ParseInt(claims.ID, 10, 64)
		if err != nil || userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
			return
		}

		sessionIDRaw := strings.TrimSpace(c.Query("sessionId"))
		if sessionIDRaw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId required"})
			return
		}
		sessionID, err := strconv.ParseInt(sessionIDRaw, 10, 64)
		if err != nil || sessionID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sessionId"})
			return
		}

		var containerID string
		err = db.QueryRow(c.Request.Context(), `
			SELECT COALESCE(target_container_id, '')
			FROM lab_sessions
			WHERE id=$1 AND user_id=$2 AND status='running'
		`, sessionID, userID).Scan(&containerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "active session not found"})
			return
		}
		if containerID == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "container not yet provisioned"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("WS terminal opened: session=%d container=%s", sessionID, containerID)

		isMock := strings.HasPrefix(containerID, "mock_")
		var cmd *exec.Cmd
		if isMock {
			shell := "sh"
			if s := os.Getenv("SHELL"); s != "" {
				shell = s
			}
			if _, err := exec.LookPath(shell); err == nil {
				cmd = exec.Command(shell)
			} else if _, err := exec.LookPath("bash"); err == nil {
				cmd = exec.Command("bash")
			} else if _, err := exec.LookPath("powershell.exe"); err == nil {
				cmd = exec.Command("powershell.exe", "-NoLogo")
			} else {
				cmd = exec.Command("cmd.exe")
			}
		} else {
			cmd = exec.Command(
				"docker", "exec", "-i", containerID,
				"sh", "-lc", "if [ -x /bin/bash ]; then exec /bin/bash; elif [ -x /bin/sh ]; then exec /bin/sh; else exec sh; fi",
			)
		}

		subStdin, err := cmd.StdinPipe()
		if err != nil {
			sendError(conn, "failed to create stdin pipe: "+err.Error())
			return
		}
		subStdout, err := cmd.StdoutPipe()
		if err != nil {
			sendError(conn, "failed to create stdout pipe: "+err.Error())
			return
		}
		cmd.Stderr = cmd.Stdout

		if err := cmd.Start(); err != nil {
			sendError(conn, "failed to start shell: "+err.Error())
			return
		}

		done := make(chan struct{})

		go func() {
			defer close(done)
			buf := make([]byte, 4096)
			for {
				n, err := subStdout.Read(buf)
				if n > 0 {
					if writeErr := conn.WriteMessage(websocket.TextMessage, buf[:n]); writeErr != nil {
						break
					}
				}
				if err != nil {
					break
				}
			}
		}()

		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					subStdin.Close()
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
					return
				}
				if _, err := io.Writer(subStdin).Write(msg); err != nil {
					return
				}
			}
		}()

		<-done
		_ = cmd.Wait()
		log.Printf("WS terminal closed: session=%d", sessionID)
	}
}

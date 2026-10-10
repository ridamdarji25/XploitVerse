package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectPostgres opens a PostgreSQL connection pool.
func ConnectPostgres(uri string) (*pgxpool.Pool, error) {
	if uri == "" {
		return nil, fmt.Errorf("POSTGRES_URI is required")
	}

	cfg, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_URI: %w", err)
	}

	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	log.Println("✅ PostgreSQL connected")
	return pool, nil
}

// RunPostgresMigrations creates required tables and indexes for the Postgres-first backend.
func RunPostgresMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'STUDENT' CHECK (role IN ('STUDENT','INSTRUCTOR','ADMIN')),
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT true,
			is_email_verified BOOLEAN NOT NULL DEFAULT false,
			last_login TIMESTAMPTZ,
			password_reset_token TEXT,
			password_reset_expires TIMESTAMPTZ,
			password_changed_at TIMESTAMPTZ,
			total_lab_time BIGINT NOT NULL DEFAULT 0,
			total_spent NUMERIC(12,2) NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_token TEXT`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_expires TIMESTAMPTZ`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_otp TEXT`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_otp_expires TIMESTAMPTZ`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_uniq ON users (LOWER(email))`,
		`CREATE INDEX IF NOT EXISTS users_password_reset_token_idx ON users (password_reset_token)`,

`CREATE TABLE IF NOT EXISTS assets (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name TEXT NOT NULL,
			source_type TEXT NOT NULL DEFAULT 'custom',
			source_ref TEXT,
			docker_image TEXT NOT NULL,
			build_context_path TEXT,
			exposed_ports_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			env_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			type TEXT NOT NULL DEFAULT 'target' CHECK (type IN ('target','attack')),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			CHECK (jsonb_typeof(exposed_ports_json) = 'array'),
			CHECK (jsonb_typeof(env_json) = 'object')
		)`,
		`ALTER TABLE assets ADD COLUMN IF NOT EXISTS build_context_path TEXT`,
		`UPDATE assets SET build_context_path = source_ref WHERE (build_context_path IS NULL OR build_context_path = '') AND source_ref IS NOT NULL AND source_ref <> ''`,

		`CREATE TABLE IF NOT EXISTS rooms (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			difficulty TEXT NOT NULL DEFAULT 'Easy' CHECK (difficulty IN ('Easy','Medium','Hard')),
			is_public BOOLEAN NOT NULL DEFAULT true,
			created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS modules (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			order_no BIGINT NOT NULL DEFAULT 1,
			points_reward BIGINT NOT NULL DEFAULT 0,
			is_published BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (room_id, order_no)
		)`,
		`CREATE INDEX IF NOT EXISTS modules_room_id_idx ON modules (room_id)`,

		`CREATE TABLE IF NOT EXISTS tasks (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			room_id BIGINT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			module_id BIGINT REFERENCES modules(id) ON DELETE SET NULL,
			asset_id BIGINT REFERENCES assets(id) ON DELETE SET NULL,
			title TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'flag' CHECK (type IN ('flag','question','interactive')),
			flag_type TEXT NOT NULL DEFAULT 'string',
			body_markdown TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '',
			hints_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			order_no BIGINT NOT NULL DEFAULT 1,
			points BIGINT NOT NULL DEFAULT 0 CHECK (points >= 0),
			hint_penalty BIGINT NOT NULL DEFAULT 0 CHECK (hint_penalty >= 0),
			flag_hash TEXT,
			is_published BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			CHECK (jsonb_typeof(hints_json) = 'array')
		)`,
		`CREATE INDEX IF NOT EXISTS tasks_room_id_idx ON tasks (room_id)`,
		`CREATE INDEX IF NOT EXISTS tasks_module_id_idx ON tasks (module_id)`,
		`CREATE INDEX IF NOT EXISTS tasks_asset_id_idx ON tasks (asset_id)`,
		`CREATE INDEX IF NOT EXISTS tasks_room_order_idx ON tasks (room_id, order_no)`,

		`CREATE TABLE IF NOT EXISTS lab_sessions (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			room_id BIGINT REFERENCES rooms(id) ON DELETE SET NULL,
			task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL,
			status TEXT NOT NULL CHECK (status IN ('pending','initializing','running','stopped','terminated','error')),
			started_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ,
			network_name TEXT,
			target_container_id TEXT,
			attack_container_id TEXT,
			connection_info_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			docker_image TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			CHECK (jsonb_typeof(connection_info_json) = 'object')
		)`,
		`CREATE INDEX IF NOT EXISTS lab_sessions_user_id_idx ON lab_sessions (user_id)`,
		`CREATE INDEX IF NOT EXISTS lab_sessions_room_id_idx ON lab_sessions (room_id)`,
		`CREATE INDEX IF NOT EXISTS lab_sessions_task_id_idx ON lab_sessions (task_id)`,
		`CREATE INDEX IF NOT EXISTS lab_sessions_status_idx ON lab_sessions (status)`,
		// Supports the per-user activity aggregate over a rolling time window.
		`CREATE INDEX IF NOT EXISTS lab_sessions_user_started_idx
			ON lab_sessions (user_id, started_at)
			WHERE started_at IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS lab_sessions_single_active_per_user
			ON lab_sessions (user_id)
			WHERE status IN ('pending','initializing','running')`,

		`CREATE TABLE IF NOT EXISTS progress (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			room_id BIGINT REFERENCES rooms(id) ON DELETE SET NULL,
			task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			state TEXT NOT NULL DEFAULT 'in_progress' CHECK (state IN ('in_progress','completed')),
			started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			completed_at TIMESTAMPTZ,
			attempts BIGINT NOT NULL DEFAULT 0,
			points_earned BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (user_id, task_id)
		)`,
		`CREATE INDEX IF NOT EXISTS progress_user_id_idx ON progress (user_id)`,
		`CREATE INDEX IF NOT EXISTS progress_task_id_idx ON progress (task_id)`,
		// Supports the per-user activity aggregate over a rolling time window.
		`CREATE INDEX IF NOT EXISTS progress_user_completed_idx
			ON progress (user_id, completed_at)
			WHERE completed_at IS NOT NULL`,
	}

	for _, stmt := range ddl {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	log.Println("✅ PostgreSQL schema ready")
	return nil
}

// SeedPostgresBaseline ensures the database has the 6 professional cybersecurity labs
// and permanently purges any deprecated basic SQLi test labs.
func SeedPostgresBaseline(ctx context.Context, pool *pgxpool.Pool) error {
	// 1. Purge deprecated basic SQLi labs if present
	_, _ = pool.Exec(ctx, `
		DELETE FROM tasks WHERE room_id IN (SELECT id FROM rooms WHERE slug IN ('sqli-lab', 'intro-sqli-room'));
		DELETE FROM modules WHERE room_id IN (SELECT id FROM rooms WHERE slug IN ('sqli-lab', 'intro-sqli-room'));
		DELETE FROM rooms WHERE slug IN ('sqli-lab', 'intro-sqli-room');
		DELETE FROM assets WHERE docker_image = 'xploitverse/sqli-lab:latest' OR source_ref = 'challenges/sqli-lab' OR name ILIKE '%basic sqli%';
	`)

	var roomCount int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM rooms`).Scan(&roomCount); err != nil {
		return fmt.Errorf("failed to count rooms: %w", err)
	}
	if roomCount >= 6 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	hashFlag := func(flag string) string {
		h := sha256.Sum256([]byte(flag))
		return hex.EncodeToString(h[:])
	}

	type taskSeed struct {
		title        string
		bodyMarkdown string
		prompt       string
		hintsJSON    string
		orderNo      int64
		points       int64
		hintPenalty  int64
		flag         string
	}

	type labSeed struct {
		slug             string
		title            string
		description      string
		difficulty       string
		category         string
		sourceRef        string
		dockerImage      string
		exposedPortsJSON string
		moduleTitle      string
		moduleDesc       string
		pointsReward     int64
		tasks            []taskSeed
	}

	labs := []labSeed{
		{
			slug:             "aws-autopsy",
			title:            "AWS Autopsy: Capital One SSRF Breach",
			description:      "Recreate the 2019 Capital One data breach ($80M fine). Exploit SSRF to steal IAM credentials via IMDSv1, then exfiltrate sensitive customer data from restricted S3 buckets.",
			difficulty:       "Hard",
			category:         "Cloud Security",
			sourceRef:        "challenges/aws-autopsy",
			dockerImage:      "xploitverse/aws-autopsy:latest",
			exposedPortsJSON: `["22/tcp","5000/tcp","8000/tcp","9000/tcp"]`,
			moduleTitle:      "AWS Infrastructure Attack Chain",
			moduleDesc:       "Step through the multi-stage AWS kill chain: SSRF, metadata extraction, S3 exfiltration, and host takeover.",
			pointsReward:     400,
			tasks: []taskSeed{
				{
					title:        "SSRF via Reverse Proxy Endpoint",
					bodyMarkdown: "# SSRF via Reverse Proxy\n\nThe web tier exposes a URL preview/proxy parameter at **http://TARGET:5000**.\n\n## Objective\nLeverage SSRF to reach the AWS Instance Metadata Service (IMDSv1) at `http://169.254.169.254`.",
					prompt:       "Exploit the proxy parameter to find the SSRF flag.",
					hintsJSON:    `["Try supplying http://169.254.169.254/latest/meta-data/ to the proxy parameter","Check the returned response body carefully"]`,
					orderNo:      1,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_aws_autopsy_ssrf_discovered}",
				},
				{
					title:        "Cloud IAM Metadata Extraction",
					bodyMarkdown: "# Cloud IAM Metadata Extraction\n\nQuery the security credentials path in the metadata service to discover the EC2 instance role.",
					prompt:       "Obtain temporary AWS security credentials from IMDSv1.",
					hintsJSON:    `["Query http://169.254.169.254/latest/meta-data/iam/security-credentials/","Extract the Role name and its token"]`,
					orderNo:      2,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_aws_autopsy_metadata_leaked}",
				},
				{
					title:        "S3 Bucket Customer Data Exfiltration",
					bodyMarkdown: "# S3 Data Exfiltration\n\nUse the stolen AWS credentials against the internal mock S3 service on port 9000 to exfiltrate customer records.",
					prompt:       "Exfiltrate the sensitive customer records from S3.",
					hintsJSON:    `["Use the AWS CLI or curl with Authorization headers targeting the S3 port","Look for the confidential customer bucket"]`,
					orderNo:      3,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_aws_autopsy_creds_stolen}",
				},
				{
					title:        "Full EC2 Instance Compromise",
					bodyMarkdown: "# EC2 Host Compromise\n\nGain root access on the target machine through lateral privilege escalation.",
					prompt:       "Retrieve the root flag at /root/root.txt.",
					hintsJSON:    `["SSH into target with student:student on port 22","Inspect background services and sudo permissions"]`,
					orderNo:      4,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_aws_autopsy_root_pwned}",
				},
			},
		},
		{
			slug:             "owasp-juice",
			title:            "OWASP Top 10 Enterprise Suite",
			description:      "Enterprise web security assessment covering Reflected XSS, Persistent Stored XSS, Insecure Direct Object References (IDOR), Server-Side Request Forgery (SSRF), and Path Traversal.",
			difficulty:       "Medium",
			category:         "Web Exploitation",
			sourceRef:        "challenges/owasp-juice",
			dockerImage:      "xploitverse/owasp-juice:latest",
			exposedPortsJSON: `["22/tcp","5000/tcp"]`,
			moduleTitle:      "OWASP Top 10 Attack Vectors",
			moduleDesc:       "Practice 5 critical vulnerability classes on a modern application target.",
			pointsReward:     500,
			tasks: []taskSeed{
				{
					title:        "Reflected Cross-Site Scripting",
					bodyMarkdown: "# Reflected XSS\n\nThe search page at **http://TARGET:5000/search** reflects user input without sanitization.",
					prompt:       "Trigger reflected XSS to retrieve the flag.",
					hintsJSON:    `["Inject a script tag into the ?q= parameter","Example: /search?q=<script>alert(1)</script>"]`,
					orderNo:      1,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_owasp_reflected_xss}",
				},
				{
					title:        "Stored XSS via Guestbook",
					bodyMarkdown: "# Stored XSS\n\nThe guestbook at **http://TARGET:5000/guestbook** persists input to the database.",
					prompt:       "Inject stored XSS payload into the guestbook.",
					hintsJSON:    `["Post a guestbook entry containing executable JavaScript","Verify the flag renders on page reload"]`,
					orderNo:      2,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_owasp_stored_xss}",
				},
				{
					title:        "Insecure Direct Object Reference (IDOR)",
					bodyMarkdown: "# IDOR Privilege Escalation\n\nThe profile page at **http://TARGET:5000/profile/2** displays user metadata.",
					prompt:       "Access administrative notes by manipulating the user ID parameter.",
					hintsJSON:    `["Change the ID in URL to 1 to inspect admin notes","The flag is embedded inside the admin profile notes"]`,
					orderNo:      3,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_owasp_idor_admin_access}",
				},
				{
					title:        "Server-Side Request Forgery (SSRF)",
					bodyMarkdown: "# SSRF to Internal Endpoints\n\nThe URL fetcher at **http://TARGET:5000/fetch** performs server-side HTTP requests.",
					prompt:       "Exploit SSRF to access the internal administrative service.",
					hintsJSON:    `["Try fetching http://127.0.0.1:5000/internal/admin","Or use file:///opt/secret/admin_key.txt"]`,
					orderNo:      4,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_owasp_ssrf_internal_access}",
				},
				{
					title:        "Path Traversal to Root Flag",
					bodyMarkdown: "# Directory Traversal\n\nThe documentation viewer at **http://TARGET:5000/read** allows path traversal.",
					prompt:       "Traverse out of the web directory to read /root/flag.txt.",
					hintsJSON:    `["Use ../ sequences in the file parameter","Count directory levels up to root"]`,
					orderNo:      5,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_owasp_path_traversal}",
				},
			},
		},
		{
			slug:             "privesc-linux",
			title:            "Linux Privilege Escalation Masterclass",
			description:      "Master advanced Linux privilege escalation across 8 real-world vectors including custom SUID binaries, cron job hijack scripts, and sudo misconfigurations.",
			difficulty:       "Hard",
			category:         "Privilege Escalation",
			sourceRef:        "challenges/privesc-linux",
			dockerImage:      "xploitverse/privesc-linux:latest",
			exposedPortsJSON: `["22/tcp"]`,
			moduleTitle:      "Kernel & Misconfiguration Privilege Escalation",
			moduleDesc:       "Enumerate Linux systems and escalate from standard user to root.",
			pointsReward:     300,
			tasks: []taskSeed{
				{
					title:        "Configuration File Credential Harvesting",
					bodyMarkdown: "# Credential Harvesting\n\nSSH in as `hacker:hacker123` on port 22. Search the filesystem for sensitive configuration files.",
					prompt:       "Find leaked database credentials in web configuration.",
					hintsJSON:    `["Inspect /opt/webapp/.env for uncommitted credentials","Look for DB_PASS"]`,
					orderNo:      1,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_privesc_config_leak}",
				},
				{
					title:        "SUID Binary Exploitation",
					bodyMarkdown: "# SUID Escalation\n\nIdentify custom SUID binaries with elevated privileges.",
					prompt:       "Abuse a custom SUID binary to escalate permissions.",
					hintsJSON:    `["Run: find / -perm -4000 -type f 2>/dev/null","Inspect /usr/local/bin/statuscheck"]`,
					orderNo:      2,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_privesc_suid_escalation}",
				},
				{
					title:        "Root Compromise via Sudo / Cron",
					bodyMarkdown: "# Root Proof Flag\n\nElevate to root and read `/root/flag.txt`.",
					prompt:       "Obtain root privileges and capture /root/flag.txt.",
					hintsJSON:    `["Inspect sudo -l and system crontabs under /etc/cron*","Extract the flag from /root/flag.txt"]`,
					orderNo:      3,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_privesc_root_owned}",
				},
			},
		},
		{
			slug:             "ssrf-lab",
			title:            "Enterprise SSRF & Cloud Metadata Exploitation",
			description:      "Real-world SSRF attack scenarios: Host header poisoning, DNS rebinding bypass, internal microservice proxy traversal, and intranet discovery.",
			difficulty:       "Hard",
			category:         "Cloud Security",
			sourceRef:        "challenges/ssrf-lab",
			dockerImage:      "xploitverse/ssrf-lab:latest",
			exposedPortsJSON: `["80/tcp"]`,
			moduleTitle:      "Advanced SSRF Exploitation",
			moduleDesc:       "Exploit server-side file fetching and host header vulnerabilities.",
			pointsReward:     100,
			tasks: []taskSeed{
				{
					title:        "SSRF File Fetch & Internal Enumeration",
					bodyMarkdown: "# Enterprise SSRF\n\nThe web application allows fetching remote assets through an unsafe file handler at **http://TARGET:80**.\n\n## Objective\nAbuse the URL parameter to read the sensitive internal flag file at `/flag.txt`.",
					prompt:       "Exploit SSRF to read /flag.txt from the server.",
					hintsJSON:    `["Try file:// URI scheme or internal localhost requests","Inspect file content fetch endpoints"]`,
					orderNo:      1,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_ssrf_vulnerable_lab_flag}",
				},
			},
		},
		{
			slug:             "reverse-shell",
			title:            "Red Team: Initial Access & Persistence",
			description:      "Execute an end-to-end Red Team intrusion: command injection for initial access foothold, SSH lateral movement, and system persistence.",
			difficulty:       "Medium",
			category:         "Red Team",
			sourceRef:        "challenges/reverse-shell",
			dockerImage:      "xploitverse/reverse-shell:latest",
			exposedPortsJSON: `["22/tcp","5000/tcp"]`,
			moduleTitle:      "Offensive Kill-Chain & Foothold",
			moduleDesc:       "Execute initial access, credential reuse, and system persistence.",
			pointsReward:     300,
			tasks: []taskSeed{
				{
					title:        "Remote Command Injection to Foothold",
					bodyMarkdown: "# Initial Access Foothold\n\nThe web service on port 5000 takes user input into a shell command.\n\n## Objective\nCatch a reverse shell or execute commands to read `/opt/webapp/flag1.txt`.",
					prompt:       "Exploit command injection to read flag1.txt.",
					hintsJSON:    `["Chain shell commands using ; or |","The flag is at /opt/webapp/flag1.txt"]`,
					orderNo:      1,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_shell_initial_foothold}",
				},
				{
					title:        "Lateral Movement via SSH Key Harvester",
					bodyMarkdown: "# Lateral Movement\n\nLocate private SSH keys left in user directories to move laterally to the `target` user.",
					prompt:       "Find the target user private key and capture flag2.txt.",
					hintsJSON:    `["Inspect ~/.ssh directories across users","Read /home/target/flag2.txt"]`,
					orderNo:      2,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_shell_lateral_movement}",
				},
				{
					title:        "Root Persistence via Cron Backdoor",
					bodyMarkdown: "# Root Persistence\n\nAbuse system cron schedules to escalate to root and verify persistence.",
					prompt:       "Capture the root persistence flag at /root/flag3.txt.",
					hintsJSON:    `["Check writable scripts executed by root in cron","Read /root/flag3.txt"]`,
					orderNo:      3,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_shell_persistence_achieved}",
				},
			},
		},
		{
			slug:             "xss-labs",
			title:            "Advanced Cross-Site Scripting (XSS) Matrix",
			description:      "Comprehensive 20-level modern XSS challenge matrix. Practice filter bypasses, script evasion, attribute injection, DOM clobbering, and CSP circumvention.",
			difficulty:       "Medium",
			category:         "Web Security",
			sourceRef:        "challenges/xss-labs",
			dockerImage:      "xploitverse/xss-labs:latest",
			exposedPortsJSON: `["80/tcp"]`,
			moduleTitle:      "XSS Matrix & Filter Evasion",
			moduleDesc:       "Master 20 progressive levels of modern Cross-Site Scripting exploitation.",
			pointsReward:     100,
			tasks: []taskSeed{
				{
					title:        "20-Stage XSS Matrix Mastery",
					bodyMarkdown: "# Advanced XSS Matrix\n\nThe web application at **http://TARGET:80** contains 20 progressive XSS challenges spanning reflected, stored, DOM, attribute injection, and filter evasion.\n\n## Objective\nBypass the sanitization filters and capture the final mastery flag.",
					prompt:       "Solve the XSS levels to submit the flag.",
					hintsJSON:    `["Level 1 begins with simple reflection in parameter name","Look for hidden form inputs and event handlers in higher levels","The flag is at /flag.txt"]`,
					orderNo:      1,
					points:       100,
					hintPenalty:  25,
					flag:         "FLAG{xv_xss_labs_mastered}",
				},
			},
		},
	}

	for _, lab := range labs {
		var assetID int64
		err := tx.QueryRow(ctx, `SELECT id FROM assets WHERE docker_image = $1 LIMIT 1`, lab.dockerImage).Scan(&assetID)
		if err != nil {
			if err := tx.QueryRow(ctx, `
				INSERT INTO assets (name, source_type, source_ref, docker_image, exposed_ports_json, env_json, type, is_active)
				VALUES ($1, 'custom', $2, $3, $4::jsonb, '{}'::jsonb, 'target', true)
				RETURNING id
			`, lab.title, lab.sourceRef, lab.dockerImage, lab.exposedPortsJSON).Scan(&assetID); err != nil {
				return fmt.Errorf("failed to seed asset for %s: %w", lab.slug, err)
			}
		}

		var roomID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO rooms (slug, title, description, difficulty, is_public)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description, difficulty = EXCLUDED.difficulty
			RETURNING id
		`, lab.slug, lab.title, lab.description, lab.difficulty).Scan(&roomID); err != nil {
			return fmt.Errorf("failed to seed room %s: %w", lab.slug, err)
		}

		var moduleID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO modules (room_id, title, description, order_no, points_reward, is_published)
			VALUES ($1, $2, $3, 1, $4, true)
			ON CONFLICT (room_id, order_no) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description, points_reward = EXCLUDED.points_reward
			RETURNING id
		`, roomID, lab.moduleTitle, lab.moduleDesc, lab.pointsReward).Scan(&moduleID); err != nil {
			return fmt.Errorf("failed to seed module for %s: %w", lab.slug, err)
		}

		for _, t := range lab.tasks {
			var existingTaskID int64
			err := tx.QueryRow(ctx, `SELECT id FROM tasks WHERE room_id = $1 AND order_no = $2 LIMIT 1`, roomID, t.orderNo).Scan(&existingTaskID)
			if err == nil {
				if _, err := tx.Exec(ctx, `
					UPDATE tasks
					SET module_id = $1, asset_id = $2, title = $3, body_markdown = $4, prompt = $5,
						hints_json = $6::jsonb, points = $7, hint_penalty = $8, flag_hash = $9, is_published = true
					WHERE id = $10
				`, moduleID, assetID, t.title, t.bodyMarkdown, t.prompt, t.hintsJSON, t.points, t.hintPenalty, hashFlag(t.flag), existingTaskID); err != nil {
					return fmt.Errorf("failed to update task %s #%d: %w", lab.slug, t.orderNo, err)
				}
			} else {
				if _, err := tx.Exec(ctx, `
					INSERT INTO tasks (room_id, module_id, asset_id, title, type, flag_type, body_markdown, prompt, hints_json, order_no, points, hint_penalty, flag_hash, is_published)
					VALUES ($1, $2, $3, $4, 'flag', 'string', $5, $6, $7::jsonb, $8, $9, $10, $11, true)
				`, roomID, moduleID, assetID, t.title, t.bodyMarkdown, t.prompt, t.hintsJSON, t.orderNo, t.points, t.hintPenalty, hashFlag(t.flag)); err != nil {
					return fmt.Errorf("failed to insert task %s #%d: %w", lab.slug, t.orderNo, err)
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	log.Println("✅ PostgreSQL baseline data seeded (6 professional labs)")
	return nil
}


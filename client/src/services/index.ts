import { apiClient } from './api';
import type {
  User,
  RegisterData,
  LoginCredentials,
  LabSession,
  ActiveSessionResponse,
  LabStartResponse,
  ProvisionResponse,
  LabSessionsListResponse,
  LabSessionDetailResponse,
  LabsListResponse,
  LabDetailResponse,
  ActivitySummary,
  CoursesListResponse,
  CourseDetailResponse,
  ModuleDetailResponse,
  TaskDetailResponse,
  LeaderboardResponse,
  MyRankResponse,
  FlagSubmitResponse,
  RegisterResponse,
  LoginResponse,
  MeResponse,
  RefreshTokenResponse,
} from '../types';

export const flagService = {
  submit: ({ taskId, flag }: { taskId: string | number; flag: string }) => {
    const normalizedTaskId =
      typeof taskId === 'string' && /^\d+$/.test(taskId.trim())
        ? Number.parseInt(taskId, 10)
        : taskId;

    return apiClient.post<FlagSubmitResponse>('/flags/submit', { taskId: normalizedTaskId, flag });
  },
};

export const authService = {
  register: (userData: RegisterData) => apiClient.post<RegisterResponse>('/auth/register', userData),
  verifyOTP: (email: string, otp: string) =>
    apiClient.post<VerifyOTPResponse>('/auth/verify-otp', { email, otp }),
  resendOTP: (email: string) =>
    apiClient.post<{ success: boolean; message: string }>('/auth/resend-otp', { email }),
  login: (credentials: LoginCredentials) => apiClient.post<LoginResponse>('/auth/login', credentials),
  logout: () => apiClient.post<{ success: boolean; message: string }>('/auth/logout'),
  getMe: () => apiClient.get<MeResponse>('/auth/me'),
  updatePassword: (passwords: { currentPassword: string; newPassword: string }) =>
    apiClient.put<{ success: boolean; message: string; data: { token: string } }>('/auth/update-password', passwords),
  refreshToken: () => apiClient.post<RefreshTokenResponse>('/auth/refresh-token'),
  forgotPassword: (email: string) =>
    apiClient.post<{ success: boolean; message: string }>('/auth/forgot-password', { email }),
  resetPassword: (token: string, password: string, confirmPassword: string) =>
    apiClient.post<{ success: boolean; message: string; data: { token: string } }>(`/auth/reset-password/${token}`, { password, confirmPassword }),
};

export const userService = {
  updateProfile: (data: Partial<User>) => apiClient.put<{ success: boolean; message: string; data: User }>('/users/profile', data),
  getMyProgress: () => apiClient.get<{ progress: Array<{ taskId: string; state: string; startedAt: string; completedAt?: string; attempts: number; pointsEarned: number }>; summary: { completedTasks: number; totalPoints: number } }>('/users/me/progress'),
  getMyActivity: () => apiClient.get<ActivitySummary>('/users/me/activity'),
};

export const labSessionService = {
  getAll: () => apiClient.get<LabSessionsListResponse>('/lab-sessions'),
  getById: (id: number) => apiClient.get<LabSessionDetailResponse>(`/lab-sessions/${id}`),
  getActive: () => apiClient.get<{ session: LabSession | null }>('/lab-sessions/active'),
  terminate: (id: number) => apiClient.post<{ success: boolean; message: string; data: { id: number; status: string } }>(`/lab-sessions/${id}/terminate`),
};

export const courseService = {
  getAll: () => apiClient.get<CoursesListResponse>('/courses'),
  getBySlug: (slug: string) => apiClient.get<CourseDetailResponse>(`/courses/${slug}`),
};

export const moduleService = {
  getById: (id: number) => apiClient.get<ModuleDetailResponse>(`/modules/${id}`),
};

export const taskService = {
  getById: (id: number) => apiClient.get<TaskDetailResponse>(`/tasks/${id}`),
};

export const labService = {
  getAll: () => apiClient.get<LabsListResponse>('/labs'),
  getById: (labId: number) => apiClient.get<LabDetailResponse>(`/labs/${labId}`),
  startLab: (labId: number) => apiClient.post<LabStartResponse>('/labs/start', { labId }),
  stopLab: (sessionId: number) => apiClient.post<{ success: boolean; message: string }>('/labs/stop', { sessionId }),
  getActiveSession: () => apiClient.get<{ session: ActiveSessionResponse | null }>('/labs/active-session'),
  completeProvisioning: (sessionId: number) => apiClient.post<ProvisionResponse>(`/labs/session/${sessionId}/provision`),
};

export const leaderboardService = {
  getTop: () => apiClient.get<LeaderboardResponse>('/leaderboard'),
  getMyRank: () => apiClient.get<MyRankResponse>('/leaderboard/me'),
};

export default {
  auth: authService,
  users: userService,
  labSessions: labSessionService,
  labs: labService,
  courses: courseService,
  modules: moduleService,
  tasks: taskService,
  flags: flagService,
  leaderboard: leaderboardService,
};
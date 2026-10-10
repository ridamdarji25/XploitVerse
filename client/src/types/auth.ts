export interface ApiResponse<T> {
  success: boolean;
  message: string;
  data?: T;
}

export interface PaginatedResponse<T> {
  items: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

export interface ErrorResponse {
  success: false;
  message: string;
}

export interface AuthTokens {
  token: string;
}

export interface User {
  id: number;
  username: string;
  email: string;
  role: 'STUDENT' | 'INSTRUCTOR' | 'ADMIN';
  firstName: string;
  lastName: string;
  fullName?: string;
  totalLabTime: number;
  totalSpent: number;
  lastLogin?: string;
  createdAt?: string;
}

export interface AuthState {
  user: User | null;
  loading: boolean;
  error: string | null;
  isAuthenticated: boolean;
  register: (userData: RegisterData) => Promise<AuthResult>;
  verifyOTP: (email: string, otp: string) => Promise<AuthResult>;
  resendOTP: (email: string) => Promise<{ success: boolean; error?: string }>;
  login: (credentials: LoginCredentials) => Promise<AuthResult>;
  logout: () => Promise<void>;
  updateUser: (userData: Partial<User>) => void;
  hasRole: (roles: string | string[]) => boolean;
  checkAuth: () => Promise<void>;
}

export interface RegisterData {
  username: string;
  email: string;
  password: string;
  firstName?: string;
  lastName?: string;
  confirmPassword?: string;
}

export interface LoginCredentials {
  email: string;
  password: string;
}

export interface AuthResult {
  success: boolean;
  user?: User;
  token?: string;
  error?: string;
  requiresVerification?: boolean;
  email?: string;
}

export interface RegisterResponse {
  email?: string;
  requiresVerification?: boolean;
  user?: User;
  token?: string;
}

export interface VerifyOTPResponse {
  user: User;
  token: string;
}

export interface LoginResponse {
  user: User;
  token: string;
}

export interface MeResponse {
  user: User;
}

export interface RefreshTokenResponse {
  token: string;
}
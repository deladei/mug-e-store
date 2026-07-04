// types/auth.types.ts

export interface User {
  id: string;
  name: string;
  email: string;
  phone: string;
  role: "customer" | "staff" | "admin";
  created_at: string;
}

export interface AuthResponse {
  access_token: string;
  user: User;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export interface RegisterPayload {
  name: string;
  email: string;
  phone: string;
  password: string;
}

export interface PasswordResetRequestPayload {
  email: string;
}

export interface PasswordResetConfirmPayload {
  token: string;
  password: string;
}

export interface MessageResponse {
  message: string;
}
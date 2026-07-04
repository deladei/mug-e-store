// services/auth.service.ts

import api, { tokenStore } from "./api";
import {
  AuthResponse,
  LoginPayload,
  RegisterPayload,
  PasswordResetRequestPayload,
  PasswordResetConfirmPayload,
  MessageResponse,
} from "@/types";

export const authService = {
  login: async (payload: LoginPayload): Promise<AuthResponse> => {
    const { data } = await api.post<AuthResponse>("/auth/login", payload);
    tokenStore.set(data.access_token);
    return data;
  },

  register: async (payload: RegisterPayload): Promise<AuthResponse> => {
    const { data } = await api.post<AuthResponse>("/auth/register", payload);
    tokenStore.set(data.access_token);
    return data;
  },

  // Called on every page load to restore session from the httpOnly cookie
  refresh: async (): Promise<AuthResponse> => {
    const { data } = await api.post<AuthResponse>("/auth/refresh");
    tokenStore.set(data.access_token);
    return data;
  },

  logout: async (): Promise<void> => {
    await api.post("/auth/logout");
    tokenStore.clear();
  },

  requestPasswordReset: async (
    payload: PasswordResetRequestPayload
  ): Promise<MessageResponse> => {
    const { data } = await api.post<MessageResponse>(
      "/auth/password-reset/request",
      payload
    );
    return data;
  },

  confirmPasswordReset: async (
    payload: PasswordResetConfirmPayload
  ): Promise<MessageResponse> => {
    const { data } = await api.post<MessageResponse>(
      "/auth/password-reset/confirm",
      payload
    );
    return data;
  },
};
import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface UserProfile {
  id: string;
  institution_id: string;
  role: string;
  full_name: string;
  email?: string;
  phone?: string;
  profile_photo_url?: string;
  permissions: string[];
}

interface AuthState {
  user: UserProfile | null;
  accessToken: string | null;
  institutionSlug: string | null;
  isAuthenticated: boolean;

  setAuth: (user: UserProfile, token: string, slug: string) => void;
  clearAuth: () => void;
  hasPermission: (perm: string) => boolean;
  hasRole: (...roles: string[]) => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      institutionSlug: null,
      isAuthenticated: false,

      setAuth: (user, accessToken, institutionSlug) => {
        // Also store in localStorage for the axios interceptor
        if (typeof window !== "undefined") {
          localStorage.setItem("access_token", accessToken);
          localStorage.setItem("institution_slug", institutionSlug);
        }
        set({ user, accessToken, institutionSlug, isAuthenticated: true });
      },

      clearAuth: () => {
        if (typeof window !== "undefined") {
          localStorage.removeItem("access_token");
          localStorage.removeItem("institution_slug");
        }
        set({ user: null, accessToken: null, institutionSlug: null, isAuthenticated: false });
      },

      hasPermission: (perm: string) => {
        const { user } = get();
        if (!user) return false;
        return user.permissions.includes(perm) || user.permissions.includes("*");
      },

      hasRole: (...roles: string[]) => {
        const { user } = get();
        if (!user) return false;
        return roles.includes(user.role);
      },
    }),
    {
      name: "institution-os-auth",
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        institutionSlug: state.institutionSlug,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

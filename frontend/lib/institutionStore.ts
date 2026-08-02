import { create } from "zustand";
import { persist } from "zustand/middleware";
import { institutionAPI } from "./api";

export interface InstitutionContext {
  institution_id: string;
  curriculum_type: "844" | "cbc";
  grading_scheme: string;
  report_card: string;
  ranking_allowed: boolean;
}

interface InstitutionState {
  context: InstitutionContext | null;
  isLoading: boolean;
  error: string | null;

  fetchContext: () => Promise<void>;
  clearContext: () => void;
}

export const useInstitutionStore = create<InstitutionState>()(
  persist(
    (set, get) => ({
      context: null,
      isLoading: false,
      error: null,

      fetchContext: async () => {
        // Avoid redundant fetches if already loaded or in flight
        if (get().context || get().isLoading) return;

        set({ isLoading: true, error: null });
        try {
          const { data } = await institutionAPI.getContext();
          set({ context: data.data, isLoading: false });
        } catch (err) {
          set({ error: "Could not load institution context.", isLoading: false });
        }
      },

      clearContext: () => set({ context: null, error: null }),
    }),
    {
      name: "institution-os-context",
      partialize: (state) => ({ context: state.context }),
    }
  )
);
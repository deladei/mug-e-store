// src/services/reports.service.ts

import api from "./api";
import { ReportsSummary } from "@/types";

export const reportsService = {
  getSummary: async (days: number): Promise<ReportsSummary> => {
    const { data } = await api.get<ReportsSummary>(
      "/admin/reports/summary",
      { params: { days } }
    );
    return data;
  },
};
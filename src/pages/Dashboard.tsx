// Main Dashboard Page Layout
// Location: ./src/pages/Dashboard.tsx

import React from 'react';
import { MainDashboard } from '@/components/Dashboard/MainDashboard';
import { TenantConfig } from '@/types/tenant';

interface DashboardProps {
  tenantId?: string;
  initialView?: 'overview' | 'requests' | 'tenant' | 'analytics';
}

export const Dashboard: React.FC<DashboardProps> = ({
  tenantId,
  initialView = 'overview',
}) => {
  return <MainDashboard initialTenantId={tenantId} />;
};

export default Dashboard;
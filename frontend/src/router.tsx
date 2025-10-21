import React from 'react'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import MainLayout from '@/components/layout/MainLayout'
import Dashboard from '@/pages/dashboard/Dashboard'
import DeviceList from '@/pages/devices/DeviceList'
import AlertList from '@/pages/alerts/AlertList'
import AuditLogList from '@/pages/audit/AuditLogList'
import Topology from '@/pages/topology/Topology'
import Operations from '@/pages/operations/Operations'

const router = createBrowserRouter([
  {
    path: '/',
    element: <MainLayout />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: 'devices',
        element: <DeviceList />,
      },
      {
        path: 'alerts',
        element: <AlertList />,
      },
      {
        path: 'audit',
        element: <AuditLogList />,
      },
      {
        path: 'topology',
        element: <Topology />,
      },
      {
        path: 'operations',
        element: <Operations />,
      },
    ],
  },
])

const AppRouter: React.FC = () => {
  return <RouterProvider router={router} />
}

export default AppRouter

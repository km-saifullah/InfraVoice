import { createBrowserRouter, Navigate } from 'react-router-dom'
import { ProtectedRoute } from './components/layout/ProtectedRoute'
import { AppLayout } from './components/layout/AppLayout'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { ProjectsListPage } from './pages/ProjectsListPage'
import { ProjectLayout } from './pages/ProjectLayout'
import { VoiceCommandPage } from './pages/VoiceCommandPage'
import { SpecificationsPage } from './pages/SpecificationsPage'
import { TerraformRunsPage } from './pages/TerraformRunsPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { path: '/', element: <Navigate to="/projects" replace /> },
          { path: '/projects', element: <ProjectsListPage /> },
          {
            path: '/projects/:projectId',
            element: <ProjectLayout />,
            children: [
              { index: true, element: <Navigate to="voice" replace /> },
              { path: 'voice', element: <VoiceCommandPage /> },
              { path: 'specifications', element: <SpecificationsPage /> },
              { path: 'terraform', element: <TerraformRunsPage /> },
            ],
          },
        ],
      },
    ],
  },
  { path: '*', element: <Navigate to="/" replace /> },
])

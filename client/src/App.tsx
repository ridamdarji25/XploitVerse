import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { AuthProvider } from './context/AuthContext';

// Layout
import Layout from './components/layout/Layout';

// Pages
import Landing from './pages/Landing';
import Login from './pages/auth/Login';
import Register from './pages/auth/Register';
import VerifyOTP from './pages/auth/VerifyOTP';
import ForgotPassword from './pages/auth/ForgotPassword';
import ResetPassword from './pages/auth/ResetPassword';
import Dashboard from './pages/dashboard/Dashboard';
import Profile from './pages/dashboard/Profile';
import Pricing from './pages/dashboard/Pricing';
import AdminDashboard from './pages/admin/AdminDashboard';
import LabWorkspace from './pages/LabWorkspace';
import CourseCatalog from './pages/courses/CourseCatalog';
import CourseDetail from './pages/courses/CourseDetail';
import ModuleDetail from './pages/courses/ModuleDetail';
import TaskDetail from './pages/courses/TaskDetail';
import Leaderboard from './pages/Leaderboard';
import NotFound from './pages/NotFound';

// Route Guards
import PrivateRoute from './components/routing/PrivateRoute';
import RoleRoute from './components/routing/RoleRoute';

function App() {
  return (
    <AuthProvider>
      <Router>
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              background: '#FBFAF6',
              color: '#1B1A17',
              border: '1px solid #DAD6CB',
              fontSize: '14px',
              borderRadius: '8px',
              boxShadow: '0 16px 40px -16px rgba(20,20,26,.18)',
            },
            success: {
              iconTheme: {
                primary: '#15A34A',
                secondary: '#FBFAF6',
              },
            },
            error: {
              iconTheme: {
                primary: '#DC2626',
                secondary: '#FBFAF6',
              },
            },
          }}
        />
        <Routes>
          {/* Public Routes */}
          <Route path="/" element={<Landing />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/verify-otp" element={<VerifyOTP />} />
          <Route path="/forgot-password" element={<ForgotPassword />} />
          <Route path="/reset-password/:token" element={<ResetPassword />} />

          {/* Protected Routes - Any authenticated user */}
          <Route element={<PrivateRoute />}>
            <Route element={<Layout />}>
              {/* Student Dashboard */}
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/pricing" element={<Pricing />} />

              {/* Course Content */}
              <Route path="/courses" element={<CourseCatalog />} />
              <Route path="/courses/:slug" element={<CourseDetail />} />
              <Route path="/modules/:id" element={<ModuleDetail />} />
              <Route path="/tasks/:id" element={<TaskDetail />} />

              {/* Leaderboard */}
              <Route path="/leaderboard" element={<Leaderboard />} />

              {/* Admin/Instructor Routes */}
              <Route element={<RoleRoute allowedRoles={['ADMIN', 'INSTRUCTOR']} />}>
                <Route path="/admin" element={<AdminDashboard />} />
              </Route>
            </Route>

            {/* Lab Workspace - Full screen without Layout */}
            <Route path="/workspace/:sessionId" element={<LabWorkspace />} />
          </Route>

          {/* 404 */}
          <Route path="*" element={<NotFound />} />
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;
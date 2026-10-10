import type { FormEvent } from 'react';
import { useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Button, Input } from '../../components/ui';
import { FadeIn } from '../../components/ui/motion';
import { AlertTriangle, ArrowRight, Shield } from 'lucide-react';

interface FormData {
  email: string;
  password: string;
}

interface FormErrors {
  email?: string;
  password?: string;
  form?: string;
}

const Login = () => {
  const savedEmail = localStorage.getItem('rememberedEmail') || '';
  const wasRemembered = Boolean(savedEmail);

  const [formData, setFormData] = useState<FormData>({
    email: savedEmail,
    password: '',
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [isLoading, setIsLoading] = useState(false);
  const [rememberMe, setRememberMe] = useState(wasRemembered);

  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const from = (location.state as { from?: { pathname: string } } | undefined)?.from?.pathname || '/dashboard';

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
    if (errors[name as keyof FormErrors]) {
      setErrors((prev) => ({ ...prev, [name]: '' }));
    }
  };

  const validateForm = () => {
    const newErrors: FormErrors = {};

    if (!formData.email) {
      newErrors.email = 'Identifier required';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = 'Invalid format';
    }

    if (!formData.password) {
      newErrors.password = 'Passkey required';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!validateForm()) return;

    setIsLoading(true);

    const result = await login(formData);

    setIsLoading(false);

    if (result.success) {
      if (rememberMe) {
        localStorage.setItem('rememberedEmail', formData.email);
      } else {
        localStorage.removeItem('rememberedEmail');
      }
      navigate(from, { replace: true });
    } else {
      if (result.requiresVerification) {
        navigate('/verify-otp', { state: { email: formData.email } });
      } else {
        setErrors((prev) => ({ ...prev, form: result.error }));
      }
    }
  };

  const handleSubmitVoid = (e: FormEvent) => {
    void handleSubmit(e);
  };

  return (
    <div className="flex min-h-[100dvh] items-center justify-center bg-bg-base p-4">
      <FadeIn className="w-full max-w-[420px]">
        {/* Brand header */}
        <div className="mb-6 text-center">
          <Link to="/" className="group mb-4 inline-flex items-center gap-2.5">
            <span className="flex h-9 w-9 items-center justify-center rounded-md bg-accent text-accent-fg">
              <Shield className="h-5 w-5" strokeWidth={1.75} />
            </span>
            <span className="text-left">
              <span className="block text-lg font-semibold leading-none tracking-tight text-fg">
                XploitVerse
              </span>
              <span className="mt-1 block text-[11px] font-medium uppercase tracking-widest text-fg-subtle">
                Auth Portal
              </span>
            </span>
          </Link>
          <h1 className="text-2xl font-semibold tracking-tight text-fg">Sign In</h1>
          <p className="mt-1 text-sm text-fg-muted">
            Authenticate your operator credentials
          </p>
        </div>

        {/* Card */}
        <div className="rounded-lg border border-border bg-bg-raised p-6 shadow-sm sm:p-7">
          <form className="space-y-4" onSubmit={handleSubmitVoid} noValidate>
            {errors.form && (
              <div
                className="flex items-center gap-2.5 rounded-md border border-danger/30 bg-danger/10 p-3 text-xs text-danger"
                role="alert"
              >
                <AlertTriangle className="h-4 w-4 shrink-0" strokeWidth={1.75} />
                <span>{errors.form}</span>
              </div>
            )}

            <Input
              label="Operative Email"
              type="email"
              name="email"
              placeholder="operative@xploitverse.io"
              value={formData.email}
              onChange={handleChange}
              error={errors.email}
              required
            />

            <Input
              label="Passkey"
              type="password"
              name="password"
              placeholder="••••••••"
              value={formData.password}
              onChange={handleChange}
              error={errors.password}
              required
            />

            <div className="flex items-center justify-between pt-0.5">
              <label className="flex cursor-pointer select-none items-center gap-2 text-xs text-fg-muted transition-colors hover:text-fg">
                <input
                  type="checkbox"
                  checked={rememberMe}
                  onChange={(e) => setRememberMe(e.target.checked)}
                  className="h-4 w-4 cursor-pointer rounded border-border-strong bg-bg-base text-accent focus:ring-accent/30"
                />
                <span>Remember session</span>
              </label>
              <Link
                to="/forgot-password"
                className="text-xs font-medium text-accent transition-colors hover:text-accent/80 hover:underline"
              >
                Forgot password?
              </Link>
            </div>

            <div className="pt-2">
              <Button
                type="submit"
                variant="primary"
                className="w-full"
                isLoading={isLoading}
              >
                Sign In
                <ArrowRight className="ml-1.5 h-4 w-4" strokeWidth={1.75} />
              </Button>
            </div>
          </form>

          <div className="mt-6 border-t border-border pt-4 text-center text-xs text-fg-muted">
            New operative?{' '}
            <Link
              to="/register"
              className="ml-1 font-semibold text-accent transition-colors hover:text-accent/80 hover:underline"
            >
              Register
            </Link>
          </div>
        </div>
      </FadeIn>
    </div>
  );
};

export default Login;

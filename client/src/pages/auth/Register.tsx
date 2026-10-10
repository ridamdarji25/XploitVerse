import type { FormEvent } from 'react';
import { useState, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Button, Input } from '../../components/ui';
import { FadeIn } from '../../components/ui/motion';
import { AlertTriangle, ArrowRight, Shield } from 'lucide-react';

interface FormData {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
  agree: boolean;
}

interface FormErrors {
  username?: string;
  email?: string;
  password?: string;
  confirmPassword?: string;
  agree?: string;
  form?: string;
}

interface TouchedFields {
  username?: boolean;
  email?: boolean;
  password?: boolean;
  confirmPassword?: boolean;
  agree?: boolean;
}

export function Register() {
  const [form, setForm] = useState<FormData>({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    agree: false,
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [touched, setTouched] = useState<TouchedFields>({});
  const [loading, setLoading] = useState(false);

  const { register } = useAuth();
  const navigate = useNavigate();

  const set = (k: keyof FormData, v: string | boolean) => {
    setForm((p) => ({ ...p, [k]: v }));
    if (errors[k as keyof FormErrors]) {
      setErrors((p) => ({ ...p, [k]: undefined }));
    }
  };
  const touch = (k: keyof TouchedFields) => setTouched((p) => ({ ...p, [k]: true }));

  const validate = useCallback(() => {
    const e: FormErrors = {};
    if (!form.username.trim()) e.username = 'Username is required';
    if (!form.email) e.email = 'Email is required';
    else if (!/\S+@\S+\.\S+/.test(form.email)) e.email = 'Invalid email format';

    if (!form.password) e.password = 'Password is required';
    else if (form.password.length < 8) e.password = 'Min. 8 chars required';

    if (!form.confirmPassword) e.confirmPassword = 'Confirmation required';
    else if (form.password !== form.confirmPassword) e.confirmPassword = 'Password mismatch';

    if (!form.agree) e.agree = 'User agreement required';
    return e;
  }, [form]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const errs = validate();
    setErrors(errs);
    setTouched({ username: true, email: true, password: true, confirmPassword: true, agree: true });

    if (Object.keys(errs).length) return;
    setLoading(true);

    const result = await register({
      username: form.username,
      email: form.email,
      password: form.password,
      confirmPassword: form.confirmPassword,
    });

    setLoading(false);

    if (result.success) {
      if (result.requiresVerification) {
        navigate('/verify-otp', { state: { email: form.email } });
      } else {
        navigate('/dashboard');
      }
    } else {
      setErrors({ form: result.error });
    }
  };

  const handleSubmitVoid = (e: FormEvent) => {
    void handleSubmit(e);
  };

  return (
    <div className="flex min-h-[100dvh] items-center justify-center bg-bg-base p-4">
      <FadeIn className="my-6 w-full max-w-[460px]">
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
                Enlistment Portal
              </span>
            </span>
          </Link>
          <h1 className="text-2xl font-semibold tracking-tight text-fg">Register Operative</h1>
          <p className="mt-1 text-sm text-fg-muted">Provision your security credentials</p>
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
              label="Operative Handle"
              type="text"
              name="username"
              placeholder="e.g. cyberwarrior"
              value={form.username}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => set('username', e.target.value)}
              onBlur={() => touch('username')}
              error={touched.username && errors.username ? errors.username : undefined}
              required
            />

            <Input
              label="Communication Email"
              type="email"
              name="email"
              placeholder="operative@xploitverse.io"
              value={form.email}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => set('email', e.target.value)}
              onBlur={() => touch('email')}
              error={touched.email && errors.email ? errors.email : undefined}
              required
            />

            <div className="grid gap-3.5 sm:grid-cols-2">
              <Input
                label="Passkey"
                type="password"
                name="password"
                placeholder="Min. 8 characters"
                value={form.password}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => set('password', e.target.value)}
                onBlur={() => touch('password')}
                error={touched.password && errors.password ? errors.password : undefined}
                required
              />

              <Input
                label="Verify Passkey"
                type="password"
                name="confirmPassword"
                placeholder="Confirm passkey"
                value={form.confirmPassword}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => set('confirmPassword', e.target.value)}
                onBlur={() => touch('confirmPassword')}
                error={touched.confirmPassword && errors.confirmPassword ? errors.confirmPassword : undefined}
                required
              />
            </div>

            <div className="pt-1">
              <label className="flex cursor-pointer select-none items-start gap-2.5 text-xs text-fg-muted transition-colors hover:text-fg">
                <input
                  type="checkbox"
                  checked={form.agree}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => set('agree', e.target.checked)}
                  className="mt-0.5 h-4 w-4 shrink-0 cursor-pointer rounded border-border-strong bg-bg-base text-accent focus:ring-accent/30"
                />
                <span className="text-[11px] leading-relaxed text-fg-muted">
                  I agree to follow operational rules and ethics policy. Intrusions outside designated challenge targets are strictly prohibited.
                </span>
              </label>
              {touched.agree && errors.agree && (
                <span className="mt-1.5 flex items-center gap-1.5 text-xs font-medium text-danger" role="alert">
                  <AlertTriangle className="h-3.5 w-3.5" strokeWidth={1.75} /> {errors.agree}
                </span>
              )}
            </div>

            <div className="pt-2">
              <Button
                type="submit"
                variant="primary"
                className="w-full"
                isLoading={loading}
              >
                Create Operative Account
                <ArrowRight className="ml-1.5 h-4 w-4" strokeWidth={1.75} />
              </Button>
            </div>
          </form>

          <div className="mt-6 border-t border-border pt-4 text-center text-xs text-fg-muted">
            Existing credentials?{' '}
            <Link
              to="/login"
              className="ml-1 font-semibold text-accent transition-colors hover:text-accent/80 hover:underline"
            >
              Sign In
            </Link>
          </div>
        </div>
      </FadeIn>
    </div>
  );
}

export default Register;

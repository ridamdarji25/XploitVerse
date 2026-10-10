import type { FormEvent } from 'react';
import { useState, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Button, Input } from '../../components/ui';
import { FadeIn } from '../../components/ui/motion';
import { AlertTriangle, CheckCircle2, RotateCcw, Shield, KeyRound } from 'lucide-react';

export function VerifyOTP() {
  const location = useLocation();
  const navigate = useNavigate();
  const { verifyOTP, resendOTP } = useAuth();

  const stateEmail = (location.state as { email?: string })?.email || '';
  const [email, setEmail] = useState(stateEmail);
  const [otp, setOtp] = useState('');
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isResending, setIsResending] = useState(false);
  const [resendCooldown, setResendCooldown] = useState(0);

  useEffect(() => {
    if (resendCooldown <= 0) return;
    const timer = setTimeout(() => setResendCooldown((prev) => prev - 1), 1000);
    return () => clearTimeout(timer);
  }, [resendCooldown]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!email.trim()) {
      setError('Email address is required');
      return;
    }
    if (!otp.trim()) {
      setError('Verification code is required');
      return;
    }

    setIsLoading(true);
    setError('');
    setInfo('');

    const res = await verifyOTP(email.trim(), otp.trim());
    setIsLoading(false);

    if (res.success) {
      navigate('/dashboard', { replace: true });
    } else {
      setError(res.error || 'Verification failed. Please check your code.');
    }
  };

  const handleResend = async () => {
    if (!email.trim() || resendCooldown > 0) return;
    setIsResending(true);
    setError('');
    setInfo('');

    const res = await resendOTP(email.trim());
    setIsResending(false);

    if (res.success) {
      setInfo('A new verification code has been transmitted to your email.');
      setResendCooldown(60);
    } else {
      setError(res.error || 'Failed to dispatch new verification code.');
    }
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
                Clearance Gate
              </span>
            </span>
          </Link>
          <h1 className="text-2xl font-semibold tracking-tight text-fg">Email Verification</h1>
          <p className="mt-1 text-sm text-fg-muted">
            Enter the 6-digit transmission code sent to your email
          </p>
        </div>

        {/* Card */}
        <div className="rounded-lg border border-border bg-bg-raised p-6 shadow-sm sm:p-7">
          <form className="space-y-4" onSubmit={(e) => void handleSubmit(e)} noValidate>
            {error && (
              <div
                className="flex items-center gap-2.5 rounded-md border border-danger/30 bg-danger/10 p-3 text-xs text-danger"
                role="alert"
              >
                <AlertTriangle className="h-4 w-4 shrink-0" strokeWidth={1.75} />
                <span>{error}</span>
              </div>
            )}

            {info && (
              <div
                className="flex items-center gap-2.5 rounded-md border border-accent/30 bg-accent/10 p-3 text-xs text-accent"
                role="status"
              >
                <CheckCircle2 className="h-4 w-4 shrink-0" strokeWidth={1.75} />
                <span>{info}</span>
              </div>
            )}

            <Input
              label="Communications Email"
              type="email"
              name="email"
              placeholder="operative@xploitverse.io"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />

            <div>
              <Input
                label="Security OTP Code"
                type="text"
                name="otp"
                placeholder="000000"
                value={otp}
                onChange={(e) => setOtp(e.target.value.replace(/\s+/g, ''))}
                className="text-center font-mono text-lg tracking-widest"
                maxLength={8}
                required
              />
              <p className="mt-1.5 text-right">
                <button
                  type="button"
                  onClick={() => void handleResend()}
                  disabled={isResending || resendCooldown > 0}
                  className="inline-flex items-center gap-1 text-xs text-accent hover:underline disabled:opacity-50"
                >
                  <RotateCcw className="h-3 w-3" />
                  {resendCooldown > 0 ? `Resend in ${resendCooldown}s` : 'Resend Code'}
                </button>
              </p>
            </div>

            <div className="pt-2">
              <Button
                type="submit"
                variant="primary"
                className="w-full"
                isLoading={isLoading}
              >
                <KeyRound className="mr-1.5 h-4 w-4" strokeWidth={1.75} />
                Verify & Activate Account
              </Button>
            </div>
          </form>

          <div className="mt-6 border-t border-border pt-4 text-center text-xs text-fg-muted">
            Already verified?{' '}
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

export default VerifyOTP;

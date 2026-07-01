interface AuthErrorProps {
  message: string;
  onDismiss?: () => void;
}

export function AuthError({ message, onDismiss }: AuthErrorProps) {
  return (
    <div className="alert">
      <span>{message}</span>
      {onDismiss && (
        <button onClick={onDismiss} className="alert__close" aria-label="Dismiss">
          ×
        </button>
      )}
    </div>
  );
}

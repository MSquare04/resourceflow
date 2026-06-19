interface ToggleSwitchProps {
  checked: boolean;
  label: string;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}

export function ToggleSwitch({ checked, label, disabled = false, onChange }: ToggleSwitchProps): JSX.Element {
  return (
    <label className={`toggle-switch ${checked ? "is-checked" : ""} ${disabled ? "is-disabled" : ""}`}>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        className="toggle-switch__control"
        disabled={disabled}
        onClick={() => onChange(!checked)}
      >
        <span className="toggle-switch__track" aria-hidden="true">
          <span className="toggle-switch__thumb" />
        </span>
      </button>
      <span className="toggle-switch__label">{label}</span>
    </label>
  );
}

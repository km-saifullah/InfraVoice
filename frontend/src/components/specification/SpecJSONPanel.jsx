import { useState } from "react";

export function SpecJSONPanel({ value, onChange, error }) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="rounded-md border border-ink-800">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-mist-300 hover:text-mist-100"
      >
        <span>{isOpen ? "Hide raw JSON" : "Edit raw JSON"}</span>
        <span className="text-mist-500">{isOpen ? "−" : "+"}</span>
      </button>
      {isOpen && (
        <div className="border-t border-ink-800 p-4">
          <textarea
            value={value}
            onChange={(event) => onChange(event.target.value)}
            spellCheck={false}
            rows={14}
            className="w-full rounded-md border border-ink-700 bg-ink-950 p-3 font-mono text-xs
              text-mist-200 focus:border-signal"
          />
          {error && <p className="mt-2 text-xs text-danger">{error}</p>}
        </div>
      )}
    </div>
  );
}

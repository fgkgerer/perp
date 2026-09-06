"use client";

import { useEffect } from "react";
import DealerDepositForm from "@/components/DealerDepositForm";

type Props = {
  open: boolean;
  onClose: () => void;
  onDeposited?: (txHash: `0x${string}`) => void;
};

export default function DealerDepositModal({ open, onClose, onDeposited }: Props) {
  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = prev;
      document.removeEventListener("keydown", onKey);
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dealer-deposit-title"
    >
      <button
        type="button"
        aria-label="关闭"
        className="absolute inset-0 bg-black/70"
        onClick={onClose}
      />
      <div className="relative z-10 max-h-[90vh] w-full overflow-y-auto rounded-t-2xl border border-panelBorder bg-panel p-5 shadow-2xl sm:max-w-md sm:rounded-2xl">
        <div className="mb-4 flex items-start justify-between gap-3">
          <div>
            <h2
              id="dealer-deposit-title"
              className="text-base font-semibold text-white"
            >
              链上保证金（Dealer.deposit）
            </h2>
            <p className="mt-1 text-xs text-muted">
              永续成交必需；与 Treasury 托管充值无关
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg px-2 py-1 text-xl leading-none text-muted hover:bg-panelBorder hover:text-white"
          >
            ×
          </button>
        </div>
        <DealerDepositForm
          compact
          onDeposited={(txHash) => {
            onDeposited?.(txHash);
            onClose();
          }}
        />
      </div>
    </div>
  );
}

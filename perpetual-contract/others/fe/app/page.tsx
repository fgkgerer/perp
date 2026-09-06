"use client";

import { useState } from "react";
import TradePage from "@/components/TradePage";
import UserProfile from "@/components/UserProfile";

export default function Page() {
  const [view, setView] = useState<"trade" | "profile">("trade");

  if (view === "profile") {
    return <UserProfile onBack={() => setView("trade")} />;
  }

  return <TradePage onOpenProfile={() => setView("profile")} />;
}

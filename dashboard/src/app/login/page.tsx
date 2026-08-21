"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";

export default function LoginPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const router = useRouter();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${process.env.NODE_ENV === 'development' ? 'http://localhost:8090/api/v1' : '/api/v1'}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });

      if (!res.ok) throw new Error("Invalid credentials");
      const data = await res.json();
      localStorage.setItem("token", data.token);
      router.push("/");
    } catch (err: any) {
      setError(err.message);
    }
  };

  return (
    <div className="flex items-center justify-center h-screen bg-black text-white">
      <form onSubmit={handleLogin} className="flex flex-col gap-4 p-8 border border-zinc-800 rounded-xl bg-zinc-950 w-96">
        <h1 className="text-2xl font-semibold mb-4 text-center">Login</h1>
        {error && <p className="text-red-500 text-sm">{error}</p>}
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          className="p-2 bg-zinc-900 border border-zinc-800 rounded text-white"
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="p-2 bg-zinc-900 border border-zinc-800 rounded text-white"
        />
        <Button type="submit" className="mt-2 bg-white text-black hover:bg-zinc-200">
          Login
        </Button>
      </form>
    </div>
  );
}

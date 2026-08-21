import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AuthCheck } from "@/components/layout/auth-check";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "pribadi-go Dashboard",
  description: "Web interface for pribadi-go local AI assistant",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className={`${inter.className} min-h-screen bg-background text-foreground antialiased`}>
        <TooltipProvider>
          <AuthCheck>
            {children}
          </AuthCheck>
        </TooltipProvider>
      </body>
    </html>
  );
}

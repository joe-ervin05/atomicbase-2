import type { Metadata } from "next";
import { Toaster } from "sonner";
import "./globals.css";

export const metadata: Metadata = {
  title: "Atombase Browser Todo",
  description: "A browser-only todo app using Atombase magic-link auth, user provisioning, and direct SDK data access.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">
        {children}
        <Toaster position="bottom-right" richColors />
      </body>
    </html>
  );
}

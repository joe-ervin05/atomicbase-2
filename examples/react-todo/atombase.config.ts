import { defineConfig } from "@atombase/cli";

export default defineConfig({
  url: process.env.NEXT_PUBLIC_ATOMBASE_URL || "http://localhost:8080",
  apiKey: process.env.ATOMBASE_API_KEY,
  definitions: "./definitions",
});

import { defineConfig } from "@atombase/cli";

export default defineConfig({
  url: process.env.ATOMBASE_URL || "http://localhost:8090",
  apiKey: process.env.ATOMBASE_API_KEY || "rM6iLhjUC+NfzmeGYNmU5C6zNrqz86zPFzFREqafdb0=",
  schemas: "./definitions",
});

import path from "path"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
    plugins: [react()],
    server: {
        watch: {
            usePolling: true,
        },
        hmr: {
            protocol: "ws",
            host: "localhost",
            port: 5173,
        },
        proxy: {
            "/api": {
                target: "http://localhost:8282",
                changeOrigin: true,
            },
        },
    },
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        },
    },
})
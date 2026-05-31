import React from "react"
import ReactDOM from "react-dom/client"
import { ColorSchemeScript, MantineProvider } from "@mantine/core"
import { ModalsProvider } from "@mantine/modals"
import { Notifications } from "@mantine/notifications"
import { BrowserRouter } from "react-router-dom"
import App from "./App"
import "@mantine/core/styles.css"
import "@mantine/notifications/styles.css"

ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
        <ColorSchemeScript defaultColorScheme="auto" />
        <MantineProvider defaultColorScheme="auto">
            <ModalsProvider>
                <Notifications position="top-right" />
                <BrowserRouter>
                    <App />
                </BrowserRouter>
            </ModalsProvider>
        </MantineProvider>
    </React.StrictMode>,
)
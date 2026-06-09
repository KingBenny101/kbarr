import React from "react"
import ReactDOM from "react-dom/client"
import { ColorSchemeScript, MantineProvider, createTheme } from "@mantine/core"
import { ModalsProvider } from "@mantine/modals"
import { Notifications } from "@mantine/notifications"
import { BrowserRouter } from "react-router-dom"
import App from "./App"
// @ts-ignore
import "@fontsource-variable/inter"
import "@mantine/core/styles.css"
import "@mantine/notifications/styles.css"

const theme = createTheme({
    fontFamily: "Inter, sans-serif",
    fontFamilyMonospace: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
})

ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
        <ColorSchemeScript defaultColorScheme="auto" />
        <MantineProvider theme={theme} defaultColorScheme="auto">
            <ModalsProvider>
                <Notifications position="top-right" />
                <BrowserRouter>
                    <App />
                </BrowserRouter>
            </ModalsProvider>
        </MantineProvider>
    </React.StrictMode>,
)
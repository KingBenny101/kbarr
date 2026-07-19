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
    colors: {
        yellow: [
            "#FFFFF0",
            "#FFFCC8",
            "#FFF78A",
            "#FFF04D",
            "#FFE620",
            "#F5D302",
            "#D4B000",
            "#AD8F00",
            "#897200",
            "#6C5900",
        ],
    },
    primaryColor: "yellow",
    primaryShade: { light: 6, dark: 5 },
    autoContrast: true,
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
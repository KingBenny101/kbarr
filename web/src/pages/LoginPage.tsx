import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Button, Card, Center, PasswordInput, Stack, Text, TextInput, Title } from "@mantine/core"
import { API_URL, setToken, showToast } from "@/utils"

export function LoginPage() {
    const [username, setUsername] = useState("")
    const [password, setPassword] = useState("")
    const [loading, setLoading] = useState(false)
    const navigate = useNavigate()

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!username || !password) return
        setLoading(true)
        try {
            const res = await fetch(`${API_URL}/api/auth/login`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password }),
            })
            if (!res.ok) {
                showToast("Invalid username or password", "error")
                return
            }
            const data = await res.json()
            setToken(data.token)
            navigate("/", { replace: true })
        } catch {
            showToast("Login failed", "error")
        } finally {
            setLoading(false)
        }
    }

    return (
        <Center h="100vh">
            <Card withBorder radius="xl" p="xl" w={360}>
                <form onSubmit={handleSubmit}>
                    <Stack gap="md">
                        <div>
                            <svg width={40} height={40} viewBox="0 0 32 32" fill="none" style={{ display: "block", margin: "0 auto 8px" }}>
                                <rect width="32" height="32" rx="7" fill="var(--mantine-color-yellow-6)"/>
                                <polygon points="16,3 26.2,7.9 28.7,18.9 21.6,27.7 10.4,27.7 3.3,18.9 5.8,7.9" fill="none" stroke="#fff" stroke-width="1.5" stroke-linejoin="round"/>
                                <polygon points="21.3,8.7 24.6,18.8 16,25 7.4,18.8 10.7,8.7" fill="none" stroke="#fff" stroke-width="1.5" stroke-linejoin="round"/>
                                <polygon points="16,11 20.3,18.5 11.7,18.5" fill="none" stroke="#fff" stroke-width="1.5" stroke-linejoin="round"/>
                            </svg>
                            <Title order={2} ta="center">kbarr</Title>
                            <Text c="dimmed" size="sm" ta="center">Sign in to continue</Text>
                        </div>
                        <TextInput
                            label="Username"
                            placeholder="Admin"
                            value={username}
                            onChange={(e) => setUsername(e.currentTarget.value)}
                            autoFocus
                        />
                        <PasswordInput
                            label="Password"
                            placeholder="••••••••"
                            value={password}
                            onChange={(e) => setPassword(e.currentTarget.value)}
                        />
                        <Button type="submit" loading={loading} fullWidth mt="xs">
                            Sign in
                        </Button>
                    </Stack>
                </form>
            </Card>
        </Center>
    )
}

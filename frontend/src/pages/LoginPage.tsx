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
                            <Title order={2}>kbarr</Title>
                            <Text c="dimmed" size="sm">Sign in to continue</Text>
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
                        <Button type="submit" color="gray" loading={loading} fullWidth mt="xs">
                            Sign in
                        </Button>
                    </Stack>
                </form>
            </Card>
        </Center>
    )
}

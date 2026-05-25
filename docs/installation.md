# Installation

Follow these steps to run kbarr locally with Docker Compose.

1. Clone the repository.

   ```bash
   git clone https://github.com/KingBenny101/kbarr.git
   cd kbarr
   ```

2. Copy `.env.example` to `.env`.

   ```bash
   cp .env.example .env
   ```

3. Open `.env` and set the variables for your environment. The defaults in `.env.example` are a good starting point for local development.

4. Start the stack.

   ```bash
   docker compose up
   ```

When the services are ready, the application will be available at `http://localhost:8282`.

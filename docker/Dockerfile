FROM codercom/code-server:latest

USER root

# Install dependencies
RUN apt-get update && apt-get install -y wget curl && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Go 1.24.3
RUN wget -q https://go.dev/dl/go1.24.3.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz \
    && rm go1.24.3.linux-amd64.tar.gz

# Install Node.js 24 via NodeSource
RUN curl -fsSL https://deb.nodesource.com/setup_24.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Claude Code globally
RUN npm install -g @anthropic-ai/claude-code

# Install git and gh CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
      | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
      | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y git gh \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Make Go available system-wide and for coder user
ENV PATH="/usr/local/go/bin:/home/coder/go/bin:$PATH"

USER coder

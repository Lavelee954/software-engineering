# Build stage - Install dependencies and compile
FROM python:3.13-slim AS builder

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PIP_NO_CACHE_DIR=1
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies required for TA-Lib and building
RUN apt-get update && apt-get install -y \
    build-essential \
    wget \
    gcc \
    g++ \
    libc6-dev \
    make \
    && rm -rf /var/lib/apt/lists/*

# Install TA-Lib C library
RUN wget http://prdownloads.sourceforge.net/ta-lib/ta-lib-0.4.0-src.tar.gz && \
    tar -xzf ta-lib-0.4.0-src.tar.gz && \
    cd ta-lib/ && \
    ./configure --prefix=/usr && \
    make && \
    make install && \
    cd .. && \
    rm -rf ta-lib ta-lib-0.4.0-src.tar.gz

# Set work directory
WORKDIR /app

# Copy requirements first for better caching
COPY requirements.txt pyproject.toml ./

# Install Python dependencies
RUN pip install --upgrade pip && \
    pip install -r requirements.txt

# Runtime stage - Minimal runtime environment
FROM python:3.13-slim AS runtime

# Add metadata
LABEL maintainer="trading-team@company.com"
LABEL description="Trading System Python 3.13 Agents - Analysis and Router"
LABEL version="1.0.0"
LABEL python.version="3.13"

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PIP_NO_CACHE_DIR=1
ENV PATH="/home/agent/.local/bin:$PATH"

# Install only runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy TA-Lib from builder stage
COPY --from=builder /usr/lib/libta_lib.* /usr/lib/
COPY --from=builder /usr/include/ta-lib/ /usr/include/ta-lib/

# Copy Python packages from builder
COPY --from=builder /usr/local/lib/python3.13/site-packages/ /usr/local/lib/python3.13/site-packages/

# Create non-root user for security
RUN useradd --create-home --shell /bin/bash --uid 1001 agent

# Set work directory
WORKDIR /app

# Copy application code with proper ownership
COPY --chown=agent:agent agents/ ./agents/
COPY --chown=agent:agent shared/ ./shared/
COPY --chown=agent:agent tests/ ./tests/
COPY --chown=agent:agent scripts/ ./scripts/
COPY --chown=agent:agent run_central_router.py ./
COPY --chown=agent:agent pyproject.toml requirements.txt ./

# Install the package in development mode
RUN pip install -e . && \
    # Create necessary directories
    mkdir -p /app/logs /app/data /app/router_data && \
    chown -R agent:agent /app

# Switch to non-root user
USER agent

# Create health check script
RUN echo '#!/bin/bash\npython -c "import agents.technical_analysis, shared.central_router; print(\"Health check passed\")"' > /app/healthcheck.sh && \
    chmod +x /app/healthcheck.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD ["/app/healthcheck.sh"] || exit 1

# Expose port for metrics
EXPOSE 8080

# Set default environment variables
ENV LOG_LEVEL=INFO
ENV PYTHONPATH=/app

# Default command (can be overridden in Kubernetes)
CMD ["python", "run_central_router.py"]
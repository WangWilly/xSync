FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY main.py .

# Expose port
EXPOSE 8001

# Set environment variables
ENV HOST=0.0.0.0
ENV PORT=8001
ENV WORKERS=1
ENV EMBEDDING_MODEL=sentence-transformers/all-MiniLM-L6-v2
ENV CROSS_ENCODER_MODEL=cross-encoder/ms-marco-MiniLM-L-6-v2

# Run the application
CMD ["python", "main.py"]

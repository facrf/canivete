FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata poppler-utils librsvg2-bin \
    imagemagick libreoffice-core libreoffice-writer libreoffice-calc \
    calibre \
    && rm -rf /var/lib/apt/lists/*

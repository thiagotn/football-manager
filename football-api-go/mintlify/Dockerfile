# Mintlify dev server — serves the API v2 documentation locally.
#
# Build context expected to be football-api-go/mintlify/.
# At runtime, the source files (mint.json, *.mdx, openapi.yaml) are mounted
# as a volume so edits hot-reload without rebuild.

FROM node:24-alpine

WORKDIR /docs

# Install Mintlify CLI globally.
RUN npm install -g mintlify@latest

EXPOSE 3001

# Bind to 0.0.0.0 so the port is reachable from the host.
CMD ["mintlify", "dev", "--port", "3001"]

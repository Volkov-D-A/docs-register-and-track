docker build -t wails-builder .

docker run --rm `
  -v "${PWD}:/src" `
  -v "${PWD}/build/bin:/out" `
  wails-builder `
  /bin/sh -c "cp -rf /src/* /app/ && rm -rf /app/frontend/node_modules /app/frontend/package-lock.json && cd /app/frontend && npm install && cd /app && wails build -o myapp && cp /app/build/bin/* /out/"
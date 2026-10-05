import multiprocessing

# Server Socket
bind = "0.0.0.0:8000"  # Bind to all interfaces on port 8000

# Worker Processes
workers = multiprocessing.cpu_count() * 2 + 1  # Number of worker processes based on CPU cores
threads = 2                                   # Number of threads per worker (useful for I/O-bound tasks)

# Worker Timeouts
timeout = 120          # Workers will timeout after 120 seconds of inactivity
graceful_timeout = 30   # Timeout for graceful worker restarts
keepalive = 5           # Keep-alive connections (in seconds)

# Logging
loglevel = "info"       # Less verbose logging for production
accesslog = "-"         # Log access requests to stdout
errorlog = "-"          # Log errors to stdout

# Preload App
preload_app = True      # Preload the app before workers are forked (saves memory)

# Security Limits
limit_request_line = 4094      # Limit the size of HTTP request lines (default: 4094)
limit_request_fields = 100     # Limit the number of HTTP headers (default: 100)
limit_request_field_size = 8190  # Limit the size of HTTP header fields (default: 8190)

# Worker Class
worker_class = "sync"   # Default synchronous workers (use async workers like "gevent" if needed)
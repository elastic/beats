import time
import sys

for x in range(0, 100):
    print(x)
    time.sleep(1)
    sys.stdout.flush()


logger = logging.getLogger('beats-logger')
total_lines = 1000
lines_per_file = 10

# Each line should have the same length + line ending
# Some spare capacity is added to make sure all events are presisted
line_length = len(str(total_lines)) + 1

# Setup python log handler
handler = logging.handlers.RotatingFileHandler(
    log_file, maxBytes=line_length * lines_per_file + 1,
    backupCount=total_lines / lines_per_file + 1)
logger.addHandler(handler)

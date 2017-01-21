import time
import sys
import logging
import logging.handlers
import socket
import random

# Writes log entries to a log file
logger = logging.getLogger('test-logger')

log_file = "/logfiles/" + socket.gethostname() + ".log"

# Setup python log handler
handler = logging.handlers.RotatingFileHandler(
    log_file, maxBytes=100000,
    backupCount=20)
logger.addHandler(handler)


# Start logging and rotating
i = 0
while True:
#for i in range(0, 10000):
    time.sleep(random.uniform(0, 0.1))
    i = i + 1
    # Tries to cause some more heavy peaks
    events = random.randrange(10) + 1
    for n in range(events):
        line = str(i) + " hello world " + str(n)
        logger.warning("%s", line)

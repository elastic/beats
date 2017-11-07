import logging
import logging.handlers
import datetime
import uuid
import os
import time
import random
import string


LOG_FILENAME = 'logs/data.log'

my_logger = logging.getLogger('filebeatLogger')
my_logger.setLevel(logging.DEBUG)

if not os.path.exists("logs"):
    os.mkdir("logs")

maxSize = 0.1 * 1000 * 1000  # 1MB
rotatedFiles = 50
logsPerSecond = 10000

handler = logging.handlers.RotatingFileHandler(
    LOG_FILENAME, maxBytes=maxSize, backupCount=rotatedFiles)
my_logger.addHandler(handler)

count = 1

sleepTime = 1.0 / logsPerSecond

while True:
    timestamp = str(datetime.datetime.now())
    length = random.randrange(100, 1000)
    randomString = ''.join(random.choice(string.ascii_uppercase + string.digits) for _ in range(length))
    log_message = timestamp + " " + str(count) + " " + str(uuid.uuid4()) + " " + randomString
    my_logger.debug(log_message)
    count = count + 1
    time.sleep(sleepTime)

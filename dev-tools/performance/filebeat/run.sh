#! /bin/bash -x

FILEBEAT_VERSION="${FILEBEAT_VERSION:=8.4.3}"
FB_TAR_NAME=filebeat-$FILEBEAT_VERSION-linux-x86_64.tar.gz
FB_FOLDER_NAME=filebeat-$FILEBEAT_VERSION-linux-x86_64
LOG_FILE="${LOG_FILE:=/tmp/flog.log}"

# Install flog
if [[ ! -e $FB_TAR_NAME ]]
then
     git clone https://github.com/belimawr/flog.git
     cd flog
     go install .
     cd ../
else
    echo "Flog folder found, assuming it's already installed"
fi

# Install Filebeat
if [ -e $FB_TAR_NAME ]
then
    echo "Filebeat already downloaded, skipping"
else
    wget https://artifacts.elastic.co/downloads/beats/filebeat/$FB_TAR_NAME
fi

if [ -e $FB_FOLDER_NAME ]
then
   echo "Filebeat already extracted"
else
    tar -xf $FB_TAR_NAME
fi

# Generate a 1Gb file
if [[ ! -e $LOG_FILE ]]
then
    flog -r 42 -n 10000000 -t log -w -o $LOG_FILE
else
    echo "Log file found"
fi

# Copy filebeat config file
cp ./filebeat.yml $FB_FOLDER_NAME
cd $FB_FOLDER_NAME

# Remove the registry
rm -rvf data

# Start filebeat
./filebeat -e -v 

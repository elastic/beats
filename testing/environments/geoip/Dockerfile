FROM ubuntu:18.04

RUN apt-get update && apt-get install -y perl build-essential

# Auto-configure CPAN
RUN cpan < /dev/null

# Install MaxMind DB writer without running tests
RUN cpan -T MaxMind::DB::Writer

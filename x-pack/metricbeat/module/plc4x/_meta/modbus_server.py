# Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
# or more contributor license agreements. Licensed under the Elastic License;
# you may not use this file except in compliance with the Elastic License.

"""Modbus TCP server with fixed values for the plc4x module integration tests.

PLC4X tag addresses are 1-based: holding-register:N reads index N-1 of the
holding list below.
"""

import struct

from pymodbus.datastore import (
    ModbusDeviceContext,
    ModbusSequentialDataBlock,
    ModbusServerContext,
)
from pymodbus.server import StartTcpServer

holding = [0] * 32
# holding-register:1:REAL is 21.5, stored big-endian in two registers.
holding[0], holding[1] = struct.unpack(">HH", struct.pack(">f", 21.5))
# holding-register:11:INT is -42.
holding[10] = struct.unpack(">H", struct.pack(">h", -42))[0]
# holding-register:12:UINT is 1234.
holding[11] = 1234

inputs = [0] * 16
# input-register:1:UINT is 777.
inputs[0] = 777

device = ModbusDeviceContext(
    co=ModbusSequentialDataBlock(1, [False] * 16),
    di=ModbusSequentialDataBlock(1, [False] * 16),
    hr=ModbusSequentialDataBlock(1, holding),
    ir=ModbusSequentialDataBlock(1, inputs),
)

StartTcpServer(context=ModbusServerContext(devices=device, single=True), address=("0.0.0.0", 5020))

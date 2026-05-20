#!/usr/bin/env python3
"""Bridge a serial port to a TCP endpoint for single-client REPL access.

This is a practical fallback when TinyGo's native ESP32-S3 WiFi netdev is not
available: the device still runs the interpreter on USB serial, and this script
exposes it on a host TCP port (default 2323).
"""

import argparse
import select
import socket
import sys

import serial


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Serial <-> TCP bridge")
    parser.add_argument("--port", required=True, help="Serial device path")
    parser.add_argument("--baud", type=int, default=115200, help="Serial baud rate")
    parser.add_argument("--host", default="0.0.0.0", help="TCP bind host")
    parser.add_argument("--tcp-port", type=int, default=2323, help="TCP bind port")
    return parser.parse_args()


def run_bridge(serial_port: str, baud: int, host: str, tcp_port: int) -> int:
    ser = serial.Serial(serial_port, baudrate=baud, timeout=0)
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind((host, tcp_port))
    server.listen(1)

    print(f"Bridge listening on {host}:{tcp_port} -> {serial_port} @ {baud}")
    print("Connect with: nc 127.0.0.1 2323")

    client = None
    try:
        while True:
            if client is None:
                conn, addr = server.accept()
                conn.setblocking(False)
                client = conn
                print(f"Client connected: {addr[0]}:{addr[1]}")

            rlist = [client]
            if ser.in_waiting > 0:
                data = ser.read(ser.in_waiting)
                if data and client is not None:
                    try:
                        client.sendall(data)
                    except OSError:
                        client.close()
                        client = None
                        print("Client disconnected")

            readable, _, _ = select.select(rlist, [], [], 0.05)
            if client in readable:
                data = client.recv(4096)
                if not data:
                    client.close()
                    client = None
                    print("Client disconnected")
                    continue
                ser.write(data)
    except KeyboardInterrupt:
        return 0
    finally:
        if client is not None:
            client.close()
        server.close()
        ser.close()


def main() -> int:
    args = parse_args()
    try:
        return run_bridge(args.port, args.baud, args.host, args.tcp_port)
    except Exception as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

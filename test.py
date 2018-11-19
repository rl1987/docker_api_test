#!/usr/bin/python

import time
import subprocess
import pynput

def wait_for_log(s):
    print('Waiting for ' + s)
    while True:
        l = proc.stdout.readline()
        l = l.decode('utf8').strip()
        print(l)
        if l.find(s) is not -1:
            return l

proc = subprocess.Popen(['./docker_api_test',
                        '-unixAddr', '/var/run/docker.sock'],
                        stdout=subprocess.PIPE,
                        stdin=subprocess.PIPE)

image_line = wait_for_log('Image: ')

container_line = wait_for_log('Created container ')

wait_for_log('Waiting')
wait_for_log('PID USER')

keyboard = Controller()
keyboard.press('q')
keyboard.release('q')

wait_for_log('Stopping container')
wait_for_log('Removing container')


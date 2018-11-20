#!/usr/bin/python

import docker
import time
import subprocess
from pynput.keyboard import Key, Controller

def wait_for_log(s):
    print('Waiting for ' + s)
    while True:
        l = proc.stdout.readline()
        l = l.decode('utf8').strip()
        print(l)
        if l.find(s) is not -1:
            return l

client = docker.from_env()

proc = subprocess.Popen(['./docker_api_test',
                        '-unixAddr', '/var/run/docker.sock'],
                        stdout=subprocess.PIPE,
                        stdin=subprocess.PIPE)

image_line = wait_for_log('Image: ')
image_id = image_line[len('Image: '):]

container_line = wait_for_log('Created container ')
container_id = container_line[len('Created container '):]

container = client.containers.get(container_id)
assert container.image.id == "sha256:" + image_id

wait_for_log('Waiting')

wait_for_log('top -')
wait_for_log('%Cpu(s):')
wait_for_log('KiB Mem :')
wait_for_log('PID USER')

keyboard = Controller()
keyboard.press('q')
keyboard.release('q')

wait_for_log('Stopping container')
wait_for_log('Removing container')


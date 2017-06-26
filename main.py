#!/usr/bin/python3
import cmd2, sys
import json
import requests
from http import cookiejar
from requests_toolbelt import MultipartEncoder, MultipartEncoderMonitor
from requests_toolbelt.utils import dump
import urllib
import logging
import progressbar
import os
import shlex
import subprocess
import optparse
import mimetypes


class FileShell(cmd2.Cmd):
    file = None
    host = "http://localhost"
    intro = 'Welcome to the gscrypto shell.  Type help or ? to list commands.\n'
    prompt = '>> '
    cookie = None
    mimetypes.init()

    def __init__(self):
        cmd2.Cmd.__init__(self)


    def do_rm(self, arg):
        if len(arg.split()) == 2 and arg.split()[0] == "-folder":
            response = requests.delete(self.host + "/auth/folder/?path=" + arg.split()[1], cookies=self.cookie)
        else:
            response = requests.delete(self.host + "/auth/file/" + arg, cookies=self.cookie)
        print(response.status_code)


    def do_login(self, arg):
        username, password = arg.split()
        response = requests.post(self.host + "/account/login", data=json.dumps({"username": username, "password": password}))

        if response.status_code != 200:
            logging.error("Invalid password")
            return

        try:
            token = response.json()["token"]
            self.cookie = {"jwt": token}
        except:
            logging.warn("Failed to get JWT token")
        else:
            logging.info("Logged in successfully")

    def do_list(self, arg):
        print(requests.get(self.host + "/auth/list/fs?path=" + arg, cookies=self.cookie).json())



    @cmd2.options([optparse.make_option('--hevc', action="store_true", help="compress with HEVC")])
    def do_upload(self, arg, opts):
        filesToUpload = []
        uploadPath, virtualfolder, tags = shlex.split(arg)
        ranSuccessfully = False
        def read_callback(monitor):
            bar.update(monitor.bytes_read)

        if os.path.isdir(uploadPath):
            for root, dirs, files in os.walk(uploadPath, topdown=False):
                for name in files:
                    filesToUpload.append(os.path.join(root, name))
        else:
            filesToUpload.append(uploadPath)

        for f in filesToUpload:
            if opts.hevc and "video" in mimetypes.guess_type(f)[0]:
                extloc = f.rfind(".")
                if extloc < 0:
                    extloc = f + ".hevc.mp4"
                else:
                    output = f[:extloc] + ".hevc.mp4"
                ret = subprocess.call(["ffmpeg", "-v", "quiet", "-stats", "-y", "-i", f, "-c:v", "libx265", "-preset", "medium", "-an", "-x265-params", "log-level=0", output])
                f = output

                if ret != 0:
                    print("failed to encode to HEVC")
                    SystemExit(1)
                else:
                    ranSuccessfully = True

            payload = {'file': (f, open(f, 'rb'), "text/plain"), "virtfolder": virtualfolder, "tags": tags}
            filesAsMultipart = MultipartEncoder(fields=payload)

            bar = progressbar.ProgressBar(widgets=[ '[', progressbar.AdaptiveTransferSpeed(), ']',
                                                     progressbar.Bar(),
                                                     progressbar.AdaptiveETA()], max_value=filesAsMultipart.len)

            m = MultipartEncoderMonitor(filesAsMultipart, read_callback)
            response = requests.post(self.host + "/auth/file/", data=m, cookies=self.cookie,headers={'Content-Type': m.content_type})
            bar.update(filesAsMultipart.len)

            if response.status_code != 201:
                print("failed: ", response.json())
            elif opts.hevc and ranSuccessfully:
                os.remove(f)

    def precmd(self, line):
        if self.cookie is None:
            raise Exception("Please login with 'login <username> <password>' first")
        return line

    def close(self):
        if self.file:
            self.file.close()
            self.file = None

if __name__ == '__main__':
    FileShell().cmdloop()

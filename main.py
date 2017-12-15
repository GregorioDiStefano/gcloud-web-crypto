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

import libtorrent as lt
import time
import tempfile
import magic

class TorrentDownloader():
    sess = None
    info = None

    def __init__(self, fn):
        self.ses = lt.session()
        self.ses.listen_on(9999, 9999)
        e = lt.bdecode(open(fn, 'rb').read())
        self.info = lt.torrent_info(e)

    def download(self, d):
        print("Downloading torrent.")

        path = d
        h = self.ses.add_torrent(self.info, path)

        while (not h.is_seed()):
            s = h.status()

            state_str = ['queued', 'checking', 'downloading metadata', \
            'downloading', 'finished', 'seeding', 'allocating', 'checking fastresume']

            sys.stdout.write('%.2f%% complete (down: %.1f kb/s up: %.1f kB/s peers: %d) %s\r' % \
                                (s.progress * 100, s.download_rate / 1000, s.upload_rate / 1000, \
                                 s.num_peers, state_str[s.state]))

            time.sleep(1)
        return path


class FileShell(cmd2.Cmd):
    file = None
    host = "http://localhost:3000"
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



    @cmd2.options([
                   optparse.make_option('--destination', type="str", help="which folder to store in remotely"),
                   optparse.make_option('--tags', type="str", help="file tags, used for searching for file in UI"),
                   optparse.make_option('--torrent-dir', type="str", help="directory to store torrent in"),
                 ])
    def do_upload(self, arg, opts):
        filesToUpload = []
        tags = shlex.split(opts.tags)
        tags = ",".join([x.strip() for x in tags])
        virtualfolder = shlex.split(opts.destination)[0]
        uploadPath = arg[0]
        print("upload path: ", uploadPath)

        def read_callback(monitor):
            bar.update(monitor.bytes_read)

        if uploadPath.endswith(".torrent"):
            if not opts.torrent_dir:
                raise Exception("please set --torrent-dir")


            td = TorrentDownloader(uploadPath)
            dlpath = shlex.split(opts.torrent_dir)[0]
            uploadPath = td.download(dlpath)
            uploadPath = os.path.relpath(uploadPath, os.getcwd())
            print("upload path: ", uploadPath)

        if os.path.isdir(uploadPath):
            for root, dirs, files in os.walk(uploadPath, topdown=False):
                for name in files:
                    filesToUpload.append(os.path.join(root, name))
        else:
            filesToUpload.append(uploadPath)

        for f in filesToUpload:
            mime = magic.Magic(mime=True)
            mt = mime.from_file(f)

            payload = {'file': (f, open(f, 'rb'), mt), "virtfolder": virtualfolder, "tags": tags}
            filesAsMultipart = MultipartEncoder(fields=payload)

            bar = progressbar.ProgressBar(widgets=[ '[', progressbar.AdaptiveTransferSpeed(), ']',
                                                     progressbar.Bar(),
                                                     progressbar.AdaptiveETA()], max_value=filesAsMultipart.len)

            m = MultipartEncoderMonitor(filesAsMultipart, read_callback)


            response = requests.post(self.host + "/auth/file/", data=m, cookies=self.cookie,headers={'Content-Type': m.content_type})
            bar.update(filesAsMultipart.len)

            if response.status_code != 201:
                print("failed: ", response.json())

    def close(self):
        if self.file:
            self.file.close()
            self.file = None

if __name__ == '__main__':
    FileShell().cmdloop()

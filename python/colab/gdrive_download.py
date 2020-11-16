#!/usr/bin/env python

import argparse
import io

from absl import app
from absl import flags
from absl import logging

from googleapiclient import discovery, http
from httplib2 import Http
from oauth2client import file, client, tools

FLAGS = flags.FLAGS

flags.DEFINE_string('auth_cache_json_file', 'storage.json',
        'The path to write a cached authorization token. It is created '
        'automatically when running the program for the first time by '
        'going through OAuth 2.0 autorization flow.')
flags.DEFINE_string('client_id_json_file', 'client_id.json',
        'The path to the JSON file with OAuth 2.0 client id and secret.'
        'You can obtain it in Google Cloud Console.')
flags.DEFINE_string('file_id', None, 'The file ID in Google Drive.')
flags.DEFINE_string('output_file', None, 'The output file name to write downloaded file.')


def main(argv):
    if len(argv) > 1:
        raise app.UsageErrors(f'Unused command line arguments: {argv[1:]}')
    if FLAGS.file_id is None:
        raise app.UsageError('Please specify --file_id.')
    if FLAGS.output_file is None:
        raise app.UsageError('Please specify --output_file.')
    SCOPES = 'https://www.googleapis.com/auth/drive.readonly'
    store = file.Storage(FLAGS.auth_cache_json_file)
    creds = store.get()
    if not creds or creds.invalid:
        if not FLAGS.client_id_json_file:
            raise app.UsageError('Please set --client_id_json_file.')
        flow = client.flow_from_clientsecrets(FLAGS.client_id_json_file, SCOPES,
                cache=None)
        creds = tools.run_flow(flow, store, flags=tools.argparser.parse_args(args=[]))
    DRIVE = discovery.build('drive', 'v3', http=creds.authorize(Http()))

    request = DRIVE.files().get_media(fileId=FLAGS.file_id)
    fh = io.BytesIO()
    downloader = http.MediaIoBaseDownload(fh, request)
    done = False
    while done is False:
        status, done = downloader.next_chunk()
        logging.info("Downloaded %d%%." % int(status.progress() * 100))
    with open(FLAGS.output_file, 'wb') as f:
        f.write(fh.getvalue())


if __name__ == '__main__':
    app.run(main)

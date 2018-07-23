#
# Copyright (c) 2016 Juniper Networks, Inc. All rights reserved.
#

import os
import subprocess


class CassandraManager(object):
    def __init__(self, cassandra_repair_logdir):
        self.cassandra_repair_logdir = cassandra_repair_logdir

    def status(self):
        subprocess.call("contrail-cassandra-status --log-file"
                        " /var/log/cassandra/status.log --debug &",
                        shell=True, close_fds=True)

    def repair(self):
        logfile = os.path.abspath(os.path.join(self.cassandra_repair_logdir,
                                               "repair.log"))
        subprocess.call("contrail-cassandra-repair --log-file"
                        " {0} --debug &".format(logfile),
                        shell=True, close_fds=True)

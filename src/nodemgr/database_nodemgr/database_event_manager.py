#
# Copyright (c) 2015 Juniper Networks, Inc. All rights reserved.
#

from gevent import monkey
monkey.patch_all()
import os
import sys
import socket
import json
import time
import datetime
import select
import gevent

from nodemgr.common.event_manager import EventManager, EventManagerTypeInfo
from nodemgr.common.cassandra_manager import CassandraManager
from pysandesh.sandesh_base import sandesh_global

from pysandesh.sandesh_session import SandeshWriter
from pysandesh.sandesh_logger import SandeshLogger
from pysandesh.gen_py.sandesh_trace.ttypes import SandeshTraceRequest
from sandesh_common.vns.ttypes import Module
from sandesh_common.vns.constants import ThreadPoolNames

from database.sandesh.database.ttypes import \
    CassandraStatusData, CassandraCompactionTask

class DatabaseEventManager(EventManager):
    def __init__(self, rule_file, unit_names, discovery_server,
                 discovery_port, collector_addr,
                 hostip, db_port, minimum_diskgb, contrail_databases,
                 cassandra_repair_interval,
                 cassandra_repair_logdir):
        self.table = "ObjectDatabaseInfo"
        if os.path.exists('/tmp/supervisord_database.sock'):
            self.supervisor_serverurl = "unix:///tmp/supervisord_database.sock"
        else:
            self.supervisor_serverurl = "unix:///var/run/supervisord_database.sock"
        type_info = EventManagerTypeInfo(
            package_name = 'contrail-database-common',
            object_table = "ObjectDatabaseInfo",
            module_type = Module.DATABASE_NODE_MGR,
            supervisor_serverurl = self.supervisor_serverurl,
            third_party_processes =  {
                "cassandra" : "Dcassandra-pidfile=.*cassandra\.pid",
                "zookeeper" : "org.apache.zookeeper.server.quorum.QuorumPeerMain"
            },
            sandesh_packages = ['database.sandesh'],
            unit_names = unit_names)
        EventManager.__init__(
            self, type_info, rule_file, discovery_server,
            discovery_port, collector_addr, sandesh_global)
        self.hostip = hostip
        self.db_port = db_port
        self.minimum_diskgb = minimum_diskgb
        self.contrail_databases = contrail_databases
        self.cassandra_repair_interval = cassandra_repair_interval
        self.cassandra_repair_logdir = cassandra_repair_logdir
        self.cassandra_mgr = CassandraManager(self.cassandra_repair_logdir,
                                              'analyticsDb',self.table,self.contrail_databases,
                                              self.hostip, self.minimum_diskgb,
                                              self.db_port)
        # Initialize tpstat structures
        self.cassandra_status_old = CassandraStatusData()
        self.cassandra_status_old.cassandra_compaction_task = CassandraCompactionTask()
        self.cassandra_status_old.thread_pool_stats = []
    # end __init__

    def _get_cassandra_config_option(self, config):
        (linux_dist, x, y) = platform.linux_distribution()
        if (linux_dist in ['Ubuntu', 'debian']):
            yamlstream = open("/etc/cassandra/cassandra.yaml", 'r')
        else:
            yamlstream = open("/etc/cassandra/conf/cassandra.yaml", 'r')

        cfg = yaml.safe_load(yamlstream)
        yamlstream.close()
        return cfg[config]

    def msg_log(self, msg, level):
        self.sandesh_global.logger().log(SandeshLogger.get_py_logger_level(
                            level), msg)

    def send_process_state_db(self, group_names):
        self.send_process_state_db_base(
            group_names, ProcessInfo)

    def send_nodemgr_process_status(self):
        self.send_nodemgr_process_status_base(
            ProcessStateNames, ProcessState, ProcessStatus)

    def get_node_third_party_process_dict(self):
        return self.third_party_process_dict 

    def get_process_state(self, fail_status_bits):
        return self.get_process_state_base(
            fail_status_bits, ProcessStateNames, ProcessState)

    def get_failbits_nodespecific_desc(self, fail_status_bits):
        description = ""
        if fail_status_bits & self.FAIL_STATUS_DISK_SPACE:
            description += "Disk for analytics db is too low," + \
                " cassandra stopped."
        if fail_status_bits & self.FAIL_STATUS_SERVER_PORT:
            if description != "":
                description += " "
            description += "Cassandra state detected DOWN."
        if fail_status_bits & self.FAIL_STATUS_DISK_SPACE_NA:
            description += "Disk space for analytics db not retrievable."
        return description

    def do_periodic_events(self):
        self.cassandra_mgr.database_periodic(self)
        self.event_tick_60()
    # end do_periodic_events

    def process(self):
        self.cassandra_mgr.process(self)

# end class DatabaseEventManager

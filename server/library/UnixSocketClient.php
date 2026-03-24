<?php

class UnixSocketClient {
    private $socketPath;
    private $conn;

    public function __construct($socketPath) {
        $this->socketPath = $socketPath;

        $this->conn = socket_create(AF_UNIX, SOCK_STREAM, 0);
        if ($this->conn === false) {
            $this->conn = null;
            return;
        }

        if (!socket_connect($this->conn, $this->socketPath)) {
            socket_close($this->conn);
            $this->conn = null;
        }
    }

    public function request($msg) {
        if ($this->conn === null) {
            return "";
        }

        $data = $msg . "\x00";

        $written = socket_write($this->conn, $data, strlen($data));
        if ($written === false) {
            return "";
        }

        $buf = socket_read($this->conn, 1024);
        if ($buf === false) {
            return "";
        }

        return $buf;
    }

    public function close() {
        if ($this->conn !== null) {
            socket_close($this->conn);
            $this->conn = null;
        }
    }
}

// -------------------- USAGE --------------------
/*
$client = new UnixSocketClient("/web/sockets/21600-LOG.sock");

$response = $client->request("UserCount");
echo $response;

$client->close();
*/

# Quick notes on using message queue

-   Use RabbitMQ to post uploaded notebooks (< 1Mb) to a queue.

-   Workers subscribe to the queue and post the reports back.

-   Server subscribes to the report queue and writes the reports to the upload
    directory.

-   For inspiration see: https://github.com/python-discord/snekbox

def infmem():
    # Allocates unbounded amount of memory.
    x = 0
    m = {}
    total = 0
    threshold = 1000
    while True:
        m[x] = "*" * x
        total += x
        x = x+1
        if total > threshold:
            print("%d bytes allocated" % total)
            threshold *= 2

import sys

from lucikit import *
from pymavlink import *

__DRONE__ = '0.0.0.0:14551'

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print 'Invalid arguments.'
        sys.exit(1)

    if sys.argv[1] == '--code':
        exec sys.argv[2]
    else:
        print 'Invalid argument.'

#!/bin/bash
echo "BeforeComment"
: <<'End'
This is a comment:
End
echo "AfterComment"
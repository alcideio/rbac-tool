 #/bin/bash -x

echo "Running UPX"
find dist/* -type f -executable -exec ./bin/upx {} +

#echo "Generate release notes footer"
echo '```sh' >  dist/notes-footer.md
dist/rbac-tool_linux_amd64/rbac-tool --help >> dist/notes-footer.md
echo '```' >>  dist/notes-footer.md
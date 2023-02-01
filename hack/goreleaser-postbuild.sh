#/bin/bash -x

if [ $2 == "windows" ]; then
  echo "Skipping Running UPX on Windows - $1"
  exit 0
fi

if [ $1 == "rbac-tool_darwin_arm64" ]; then
  echo "Skipping Running UPX on Darwin arm64 - $1"
  exit 0
fi

if [ $1 == "rbac-tool_darwin_amd64" ]; then
  echo "Skipping Running UPX on Darwin amd64 - $1"
  exit 0
fi

echo "Running UPX - $1"
find dist/$1* -type f -executable -exec ./bin/upx {} +

#echo "Generate release notes footer"
echo '```sh' >  dist/notes-footer.md
if [ $1 == "rbac-tool_linux_amd64" ]; then
  dist/rbac-tool_linux_amd64/rbac-tool --help >> dist/notes-footer.md
fi
echo '```' >>  dist/notes-footer.md
int main() {
  try {
    throw 42;
  } catch(int thing) {
    return thing;
  }
  return 0;
}
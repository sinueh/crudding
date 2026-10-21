from setuptools import find_packages, setup

setup(
    name="python-raw-flat",
    package_dir={"": "src"},
    packages=find_packages(where="src"),
)
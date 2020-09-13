FROM fedora as builder

RUN dnf -y install python gcc gcc-c++ boost cmake boost-python2 OpenImageIO OpenImageIO-devel python2-pyside python2-pyopengl pyside-tools vim python-pyside-devel python2-jinja2 python2-pthreading python-pyside-devel 

WORKDIR /root/

COPY /USD.tgz ./USD.tgz
COPY /build-usd-with-cleanup.sh ./build-usd-with-cleanup.sh

RUN tar -xzvf USD.tgz
RUN sh build-usd-with-cleanup.sh

FROM fedora

RUN dnf -y install python zip

WORKDIR /root/

COPY /android-tools.tgz /root/android-tools.tgz
COPY --from=0 /usr/local/USD /usr/local/USD

RUN tar -xzvf ./android-tools.tgz && rm android-tools.tgz

ENV PYTHONPATH=$PYTHONPATH:/usr/local/USD/lib/python/
ENV PATH=$PATH:/root/android-tools/27.0.3/

CMD ["bash"]

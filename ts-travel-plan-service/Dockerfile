FROM python:3.11.11-bookworm AS build
ARG PIP_INDEX_URL=https://pypi.org/simple
ARG PIP_EXTRA_INDEX_URL
ARG PIP_FIND_LINKS
ARG PIP_NO_INDEX
ARG PIP_TRUSTED_HOST
WORKDIR /build
COPY requirements.lock build-requirements.lock ./
RUN python -m venv /opt/venv \
    && /opt/venv/bin/pip install --require-hashes -r build-requirements.lock \
    && /opt/venv/bin/pip install --no-build-isolation --require-hashes -r requirements.lock \
    && /opt/venv/bin/pip check

FROM python:3.11.11-slim-bookworm
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    TZ=Asia/Shanghai \
    PATH=/opt/venv/bin:$PATH
WORKDIR /app
RUN addgroup --system app \
    && adduser --system --ingroup app app
COPY --from=build /opt/venv /opt/venv
COPY app ./app
USER app
EXPOSE 14322
CMD ["/opt/venv/bin/uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "14322", "--no-access-log"]

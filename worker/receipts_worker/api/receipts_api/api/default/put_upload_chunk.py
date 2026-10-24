from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.upload_offset import UploadOffset
from ...types import File, FileTypes
from io import BytesIO
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: File,
    content_range: str,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Content-Range"] = content_range

    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "put",
        "url": "/api/uploads/{id}/chunks".format(id=quote(str(id), safe=""),),
    }

    _kwargs["content"] = body.payload
    headers["Content-Type"] = "application/octet-stream"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | UploadOffset:
    if response.status_code == 200:
        response_200 = UploadOffset.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | UploadOffset]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: File,
    content_range: str,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | UploadOffset]:
    """
    Args:
        id (str):
        content_range (str):
        origin (str):
        x_csrf_token (str):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | UploadOffset]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
content_range=content_range,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: File,
    content_range: str,
    origin: str,
    x_csrf_token: str,

) -> Error | UploadOffset | None:
    """
    Args:
        id (str):
        content_range (str):
        origin (str):
        x_csrf_token (str):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | UploadOffset
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
content_range=content_range,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: File,
    content_range: str,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | UploadOffset]:
    """
    Args:
        id (str):
        content_range (str):
        origin (str):
        x_csrf_token (str):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | UploadOffset]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
content_range=content_range,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    body: File,
    content_range: str,
    origin: str,
    x_csrf_token: str,

) -> Error | UploadOffset | None:
    """
    Args:
        id (str):
        content_range (str):
        origin (str):
        x_csrf_token (str):
        body (File):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | UploadOffset
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
content_range=content_range,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed

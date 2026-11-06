from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_registration import WorkerRegistration
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: str,
    *,
    allow_insecure_install: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token





    params: dict[str, Any] = {}

    params["allow_insecure_install"] = allow_insecure_install


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/workers/{id}/token".format(id=quote(str(id), safe=""),),
        "params": params,
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerRegistration:
    if response.status_code == 201:
        response_201 = WorkerRegistration.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerRegistration]:
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
    allow_insecure_install: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerRegistration]:
    """  Administrateurs uniquement. Refusé (404) pour un worker révoqué.

    Args:
        id (str):
        allow_insecure_install (bool | Unset):  Default: False.
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerRegistration]
     """


    kwargs = _get_kwargs(
        id=id,
allow_insecure_install=allow_insecure_install,
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
    allow_insecure_install: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerRegistration | None:
    """  Administrateurs uniquement. Refusé (404) pour un worker révoqué.

    Args:
        id (str):
        allow_insecure_install (bool | Unset):  Default: False.
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerRegistration
     """


    return sync_detailed(
        id=id,
client=client,
allow_insecure_install=allow_insecure_install,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    allow_insecure_install: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | WorkerRegistration]:
    """  Administrateurs uniquement. Refusé (404) pour un worker révoqué.

    Args:
        id (str):
        allow_insecure_install (bool | Unset):  Default: False.
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerRegistration]
     """


    kwargs = _get_kwargs(
        id=id,
allow_insecure_install=allow_insecure_install,
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
    allow_insecure_install: bool | Unset = False,
    origin: str,
    x_csrf_token: str,

) -> Error | WorkerRegistration | None:
    """  Administrateurs uniquement. Refusé (404) pour un worker révoqué.

    Args:
        id (str):
        allow_insecure_install (bool | Unset):  Default: False.
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerRegistration
     """


    return (await asyncio_detailed(
        id=id,
client=client,
allow_insecure_install=allow_insecure_install,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed

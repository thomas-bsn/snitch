from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.get_worker_update_response_200 import GetWorkerUpdateResponse200
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/update",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | GetWorkerUpdateResponse200:
    if response.status_code == 200:
        response_200 = GetWorkerUpdateResponse200.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | GetWorkerUpdateResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | GetWorkerUpdateResponse200]:
    """  Image que l'agent local du Worker doit faire tourner, et si ce Worker est au repos (aucun job sous
    bail) pour que l'agent attende un moment calme. En mode latest, résout la cible depuis le catalogue
    en cache (écriture seulement si une version plus haute apparaît). Sans cible, les champs version et
    image sont nuls.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetWorkerUpdateResponse200]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | GetWorkerUpdateResponse200 | None:
    """  Image que l'agent local du Worker doit faire tourner, et si ce Worker est au repos (aucun job sous
    bail) pour que l'agent attende un moment calme. En mode latest, résout la cible depuis le catalogue
    en cache (écriture seulement si une version plus haute apparaît). Sans cible, les champs version et
    image sont nuls.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetWorkerUpdateResponse200
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | GetWorkerUpdateResponse200]:
    """  Image que l'agent local du Worker doit faire tourner, et si ce Worker est au repos (aucun job sous
    bail) pour que l'agent attende un moment calme. En mode latest, résout la cible depuis le catalogue
    en cache (écriture seulement si une version plus haute apparaît). Sans cible, les champs version et
    image sont nuls.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetWorkerUpdateResponse200]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | GetWorkerUpdateResponse200 | None:
    """  Image que l'agent local du Worker doit faire tourner, et si ce Worker est au repos (aucun job sous
    bail) pour que l'agent attende un moment calme. En mode latest, résout la cible depuis le catalogue
    en cache (écriture seulement si une version plus haute apparaît). Sans cible, les champs version et
    image sont nuls.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetWorkerUpdateResponse200
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
